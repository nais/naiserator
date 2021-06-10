package synchronizer

import (
	"context"
	"fmt"
	"sync"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileNaisjob process Naisjob work queue
func (n *Synchronizer) ReconcileNaisjob(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), n.Config.Synchronizer.SynchronizationTimeout)
	defer cancel()

	naisjob := &nais_io_v1.Naisjob{}
	err := n.Get(ctx, req.NamespacedName, naisjob)
	if err != nil {
		if errors.IsNotFound(err) {
			logger := log.WithFields(log.Fields{
				"namespace": req.Namespace,
				"naisjob":   req.Name,
			})
			logger.Infof("Naisjob has been deleted from Kubernetes")

			err = nil
		}
		return ctrl.Result{}, err
	}

	changed := true

	logger := *log.WithFields(naisjob.LogFields())

	// Update Naisjob resource with cronjob/job event
	defer func() {
		if !changed {
			return
		}
		err := n.UpdateNaisjob(ctx, naisjob, func(existing *nais_io_v1.Naisjob) error {
			existing.Status = naisjob.Status
			return n.Update(ctx, naisjob)
		})
		if err != nil {
			n.reportError(ctx, EventFailedStatusUpdate, err, naisjob)
		} else {
			logger.Debugf("Naisjob status: %+v'", naisjob.Status)
		}
	}()

	rollout, err := n.PrepareNaisjob(naisjob)
	if err != nil {
		naisjob.Status.SynchronizationState = EventFailedPrepare
		n.reportError(ctx, naisjob.Status.SynchronizationState, err, naisjob)
		return ctrl.Result{RequeueAfter: prepareRetryInterval}, nil
	}

	if rollout == nil {
		changed = false
		logger.Debugf("Naisjob synchronization hash not changed; skipping synchronization")
		return ctrl.Result{}, nil
	}

	logger = *log.WithFields(naisjob.LogFields())
	logger.Debugf("Starting synchronization")
	metrics.NaisjobsProcessed.Inc()

	naisjob.Status.CorrelationID = rollout.CorrelationID

	err, retry := n.Sync(ctx, *rollout)
	if err != nil {
		if retry {
			naisjob.Status.SynchronizationState = EventRetrying
			metrics.NaisjobsRetries.Inc()
			n.reportError(ctx, naisjob.Status.SynchronizationState, err, naisjob)
		} else {
			naisjob.Status.SynchronizationState = EventFailedSynchronization
			naisjob.Status.SynchronizationHash = rollout.SynchronizationHash // permanent failure
			metrics.NaisjobsFailed.Inc()
			n.reportError(ctx, naisjob.Status.SynchronizationState, err, naisjob)
			err = nil
		}
		return ctrl.Result{}, err
	}

	// Synchronization OK
	logger.Debugf("Successful synchronization")
	naisjob.Status.SynchronizationState = EventSynchronized
	naisjob.Status.SynchronizationHash = rollout.SynchronizationHash
	naisjob.Status.SynchronizationTime = time.Now().UnixNano()
	metrics.NaisjobsDeployments.Inc()

	_, err = n.reportEvent(ctx, resource.CreateEvent(naisjob, naisjob.Status.SynchronizationState, "Successfully synchronized all naisjob resources", "Normal"))
	if err != nil {
		log.Errorf("While creating an event for this rollout, an error occurred: %s", err)
	}

	// Create new NAIS deployment event
	event := generator.NewDeploymentEvent(naisjob, naisjob.Spec.Image)
	naisjob.SetDeploymentRolloutStatus(event.RolloutStatus.String())

	// Monitor the rollout status so that we can report a successfully completed rollout to NAIS deploy.
	go n.MonitorRollout(naisjob, logger, n.Config.Synchronizer.RolloutCheckInterval, n.Config.Synchronizer.RolloutTimeout, naisjob.Spec.Image)
	return ctrl.Result{}, nil
}

// PrepareNaisjob converts a NAIS Naisjob spec into a Rollout object.
// This is a read-only operation
// The Rollout object contains callback functions that commits changes in the cluster.
func (n *Synchronizer) PrepareNaisjob(naisjob *nais_io_v1.Naisjob) (*Rollout, error) {
	ctx := context.Background()
	var err error

	rollout := &Rollout{
		Source:          naisjob,
		ResourceOptions: n.ResourceOptions,
	}

	if err = naisjob.ApplyDefaults(); err != nil {
		return nil, fmt.Errorf("BUG: merge default values into naisjob: %s", err)
	}

	rollout.SynchronizationHash, err = naisjob.Hash()
	if err != nil {
		return nil, fmt.Errorf("BUG: create naisjob hash: %s", err)
	}

	// Skip processing if naisjob didn't change since last synchronization.
	if naisjob.Status.SynchronizationHash == rollout.SynchronizationHash {
		return nil, nil
	}

	err = naisjob.EnsureCorrelationID()
	if err != nil {
		return nil, err
	}

	rollout.CorrelationID = naisjob.CorrelationID()

	// Retrieve current namespace to check for labels and annotations
	namespace := &corev1.Namespace{}
	err = n.Get(ctx, client.ObjectKey{Name: naisjob.GetNamespace()}, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing namespace: %s", err)
	}

	// Assert that CNRM annotations are set on namespaces when CNRM support is enabled
	if naisjob.Spec.GCP != nil && (naisjob.Spec.GCP.SqlInstances != nil || naisjob.Spec.GCP.Permissions != nil) {
		if val, ok := namespace.Annotations["cnrm.cloud.google.com/project-id"]; ok {
			rollout.SetGoogleTeamProjectId(val)
		} else {
			return nil, fmt.Errorf("GCP resources requested, but no team project ID annotation set on namespace %s (not running on GCP?)", naisjob.GetNamespace())
		}
	}

	// Create Linkerd resources only if feature is enabled and namespace is Linkerd-enabled
	if n.Config.Features.Linkerd && namespace.Annotations["linkerd.io/inject"] == "enabled" {
		rollout.ResourceOptions.Linkerd = true
	}

	rollout.ResourceOperations, err = resourcecreator.CreateNaisjob(naisjob, rollout.ResourceOptions)

	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}

var naisjobsync sync.Mutex

// UpdateNaisjob atomically update an Naisjob resource.
// Locks the resource to avoid race conditions.
func (n *Synchronizer) UpdateNaisjob(ctx context.Context, source resource.Source, updateFunc func(existing *nais_io_v1.Naisjob) error) error {
	naisjobsync.Lock()
	defer naisjobsync.Unlock()

	existing := &nais_io_v1.Naisjob{}
	err := n.Get(ctx, client.ObjectKey{Namespace: source.GetNamespace(), Name: source.GetName()}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of Naisjob: %s", err)
	}

	return updateFunc(existing)
}
