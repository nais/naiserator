package synchronizer

import (
	"context"
	"fmt"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
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
func (n *Synchronizer) ReconcileNaisjob(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, n.Config.Synchronizer.SynchronizationTimeout)
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
		err := n.UpdateResource(ctx, naisjob, func(existing resource.Source) error {
			existing.SetStatus(naisjob.GetStatus())
			return n.Update(ctx, existing) // was naisjob
		})
		if err != nil {
			n.reportError(ctx, EventFailedStatusUpdate, err, naisjob)
		} else {
			logger.Debugf("Naisjob status: %+v'", naisjob.Status)
		}
	}()

	rollout, err := n.PrepareNaisjob(ctx, naisjob)
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

	retry, err := n.Sync(ctx, *rollout)
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

	return ctrl.Result{}, nil
}

// PrepareNaisjob converts a NAIS Naisjob spec into a Rollout object.
// This is a read-only operation
// The Rollout object contains callback functions that commits changes in the cluster.
func (n *Synchronizer) PrepareNaisjob(ctx context.Context, naisjob *nais_io_v1.Naisjob) (*Rollout, error) {
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

	if naisjob.Spec.GCP != nil {
		// App requests gcp resources, verify we've got a GCP team project ID
		projectID, ok := namespace.Annotations["cnrm.cloud.google.com/project-id"]
		if !ok {
			// We're not currently in a team namespace with corresponding GCP team project
			return nil, fmt.Errorf("GCP resources requested, but no team project ID annotation set on namespace %s (not running on GCP?)", naisjob.GetNamespace())
		}
		rollout.ResourceOptions.GoogleTeamProjectId = projectID
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
