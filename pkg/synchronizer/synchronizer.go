package synchronizer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	naiserator_scheme "github.com/nais/naiserator/pkg/scheme"
	"github.com/nais/naiserator/updater"
)

// Machine readable event "Reason" fields, used for determining deployment state.
const (
	EventSynchronized          = "Synchronized"
	EventRolloutComplete       = "RolloutComplete"
	EventFailedPrepare         = "FailedPrepare"
	EventFailedSynchronization = "FailedSynchronization"
	EventFailedStatusUpdate    = "FailedStatusUpdate"
	EventRetrying              = "Retrying"
)

const (
	prepareRetryInterval = time.Minute * 30
)

// Synchronizer creates child resources from Application resources in the cluster.
// If the child resources does not match the Application spec, the resources are updated.
type Synchronizer struct {
	client.Client
	RolloutMonitor  map[client.ObjectKey]RolloutMonitor
	SimpleClient    client.Client
	Scheme          *runtime.Scheme
	ResourceOptions resource.Options
	Config          config.Config
	Kafka           kafka.Interface
}

// Commit wraps a cluster operation function with extra fields
type commit struct {
	groupVersionKind schema.GroupVersionKind
	fn               func() error
}

// Creates a Kubernetes event, or updates an existing one with an incremented counter
func (n *Synchronizer) reportEvent(ctx context.Context, reportedEvent *corev1.Event) (*corev1.Event, error) {
	selector, err := fields.ParseSelector(fmt.Sprintf("involvedObject.name=%s,involvedObject.uid=%s", reportedEvent.InvolvedObject.Name, reportedEvent.InvolvedObject.UID))
	if err != nil {
		return nil, fmt.Errorf("internal error: unable to parse query: %s", err)
	}
	events := &corev1.EventList{}
	err = n.SimpleClient.List(ctx, events, &client.ListOptions{
		FieldSelector: selector,
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("get events for app '%s': %s", reportedEvent.InvolvedObject.Name, err)
	}

	for _, event := range events.Items {
		if event.Message == reportedEvent.Message {
			event.Count++
			event.LastTimestamp = reportedEvent.LastTimestamp
			event.SetAnnotations(reportedEvent.GetAnnotations())
			return &event, n.Update(ctx, &event)
		}
	}

	err = n.Create(ctx, reportedEvent)
	if err != nil {
		return nil, err
	}
	return reportedEvent, nil
}

// Reports an error through the error log, a Kubernetes event, and possibly logs a failure in event creation.
func (n *Synchronizer) reportError(ctx context.Context, eventSource string, err error, source resource.Source) {
	logger := log.WithFields(source.LogFields())
	logger.Error(err)
	_, err = n.reportEvent(ctx, resource.CreateEvent(source, eventSource, err.Error(), "Warning"))
	if err != nil {
		logger.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

// ReconcileApplication process Application work queue
func (n *Synchronizer) ReconcileApplication(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), n.Config.Synchronizer.SynchronizationTimeout)
	defer cancel()

	app := &nais_io_v1alpha1.Application{}
	err := n.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			logger := log.WithFields(log.Fields{
				"namespace":   req.Namespace,
				"application": req.Name,
			})
			logger.Infof("Application has been deleted from Kubernetes")

			err = nil
		}
		return ctrl.Result{}, err
	}

	changed := true

	logger := *log.WithFields(app.LogFields())

	// Update Application resource with deployment event
	defer func() {
		if !changed {
			return
		}
		err := n.UpdateApplication(ctx, app, func(existing *nais_io_v1alpha1.Application) error {
			existing.Status = app.Status
			return n.Update(ctx, app)
		})
		if err != nil {
			n.reportError(ctx, EventFailedStatusUpdate, err, app)
		} else {
			logger.Debugf("Application status: %+v'", app.Status)
		}
	}()

	rollout, err := n.Prepare(app)
	if err != nil {
		app.Status.SynchronizationState = EventFailedPrepare
		n.reportError(ctx, app.Status.SynchronizationState, err, app)
		return ctrl.Result{RequeueAfter: prepareRetryInterval}, nil
	}

	if rollout == nil {
		changed = false
		logger.Debugf("Synchronization hash not changed; skipping synchronization")

		// Application is not rolled out completely; start monitoring
		if app.Status.SynchronizationState == EventSynchronized {
			n.MonitorRollout(app, logger)
		}

		return ctrl.Result{}, nil
	}

	logger = *log.WithFields(app.LogFields())
	logger.Debugf("Starting synchronization")
	metrics.ApplicationsProcessed.Inc()

	app.Status.CorrelationID = rollout.CorrelationID

	retry, err := n.Sync(ctx, *rollout)
	if err != nil {
		if retry {
			app.Status.SynchronizationState = EventRetrying
			metrics.ApplicationsRetries.Inc()
			n.reportError(ctx, app.Status.SynchronizationState, err, app)
		} else {
			app.Status.SynchronizationState = EventFailedSynchronization
			app.Status.SynchronizationHash = rollout.SynchronizationHash // permanent failure
			metrics.ApplicationsFailed.Inc()
			n.reportError(ctx, app.Status.SynchronizationState, err, app)
			err = nil
		}
		return ctrl.Result{}, err
	}

	// Synchronization OK
	logger.Debugf("Successful synchronization")
	app.Status.SynchronizationState = EventSynchronized
	app.Status.SynchronizationHash = rollout.SynchronizationHash
	app.Status.SynchronizationTime = time.Now().UnixNano()
	metrics.Deployments.Inc()

	_, err = n.reportEvent(ctx, resource.CreateEvent(app, app.Status.SynchronizationState, "Successfully synchronized all application resources", "Normal"))
	if err != nil {
		log.Errorf("While creating an event for this rollout, an error occurred: %s", err)
	}

	// Monitor the rollout status so that we can report a successfully completed rollout to NAIS deploy.
	n.MonitorRollout(app, logger)

	return ctrl.Result{}, nil
}

// Unreferenced return all resources in cluster which was created by synchronizer previously, but is not included in the current rollout.
func (n *Synchronizer) Unreferenced(ctx context.Context, rollout Rollout) ([]runtime.Object, error) {
	// Return true if a cluster resource also is applied with the rollout.
	intersects := func(existing runtime.Object) bool {
		existingMeta, err := meta.Accessor(existing)
		if err != nil {
			log.Errorf("BUG: unable to determine TypeMeta for existing resource: %s", err)
			return true
		}
		for _, rop := range rollout.ResourceOperations {
			// Normally we would use GroupVersionKind to compare resource types, but due to
			// https://github.com/kubernetes/client-go/issues/308 the GVK is not set on the existing resource.
			// Reflection seems to work fine here.
			resourceMeta, err := meta.Accessor(rop.Resource)
			if err != nil {
				log.Errorf("BUG: unable to determine TypeMeta for new resource: %s", err)
				return true
			}
			if reflect.TypeOf(rop.Resource) == reflect.TypeOf(existing) {
				if resourceMeta.GetName() == existingMeta.GetName() {
					return true
				}
			}
		}
		return false
	}

	listers := naiserator_scheme.GenericListers()
	if len(n.ResourceOptions.GoogleProjectId) > 0 {
		listers = append(listers, naiserator_scheme.GCPListers()...)
	}

	if n.ResourceOptions.Istio {
		listers = append(listers, naiserator_scheme.IstioListers()...)
	}

	if n.ResourceOptions.AzureServiceOperatorEnabled {
		listers = append(listers, naiserator_scheme.ASOListers()...)
	}
	resources, err := updater.FindAll(ctx, n, n.Scheme, listers, rollout.Source)
	if err != nil {
		return nil, fmt.Errorf("discovering unreferenced resources: %s", err)
	}

	unreferenced := make([]runtime.Object, 0, len(resources))
	for _, existing := range resources {
		if !intersects(existing) {
			unreferenced = append(unreferenced, existing)
		}
	}

	return unreferenced, nil
}

func (n *Synchronizer) rolloutWithRetryAndMetrics(commits []commit) (bool, error) {
	for _, commit := range commits {
		if err := observeDuration(commit.fn); err != nil {
			retry := false
			// In case of race condition errors
			if errors.IsConflict(err) {
				retry = true
			}
			reason := errors.ReasonForError(err)
			return retry, fmt.Errorf("persisting resource to Kubernetes: %s: %s", reason, err)
		}
		metrics.ResourcesGenerated.WithLabelValues(commit.groupVersionKind.Kind).Inc()
	}
	return false, nil
}

func (n *Synchronizer) Sync(ctx context.Context, rollout Rollout) (bool, error) {
	commits := n.ClusterOperations(ctx, rollout)
	return n.rolloutWithRetryAndMetrics(commits)
}

// Prepare converts a NAIS application spec into a Rollout object.
// This is a read-only operation
// The Rollout object contains callback functions that commits changes in the cluster.
func (n *Synchronizer) Prepare(app *nais_io_v1alpha1.Application) (*Rollout, error) {
	ctx := context.Background()
	var err error

	rollout := &Rollout{
		Source:          app,
		ResourceOptions: n.ResourceOptions,
	}

	if err = app.ApplyDefaults(); err != nil {
		return nil, fmt.Errorf("BUG: merge default values into application: %s", err)
	}

	rollout.SynchronizationHash, err = app.Hash()
	if err != nil {
		return nil, fmt.Errorf("BUG: create application hash: %s", err)
	}

	// Skip processing if application didn't change since last synchronization.
	if app.Status.SynchronizationHash == rollout.SynchronizationHash {
		return nil, nil
	}

	err = app.EnsureCorrelationID()
	if err != nil {
		return nil, err
	}

	rollout.CorrelationID = app.CorrelationID()

	// Make a query to Kubernetes for this application's previous deployment.
	// The number of replicas is significant, so we need to carry it over to match
	// this next rollout.

	//TODO: Skatt we do not need this.
	previousDeployment := &apps.Deployment{}
	err = n.Get(ctx, client.ObjectKey{Name: app.GetName(), Namespace: app.GetNamespace()}, previousDeployment)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing deployment: %s", err)
	}

	// Retrieve current namespace to check for labels and annotations
	namespace := &corev1.Namespace{}
	err = n.Get(ctx, client.ObjectKey{Name: app.GetNamespace()}, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing namespace: %s", err)
	}

	if app.Spec.GCP != nil {
		// App requests gcp resources, verify we've got a GCP team project ID
		projectID, ok := namespace.Annotations["cnrm.cloud.google.com/project-id"]
		if !ok {
			// We're not currently in a team namespace with corresponding GCP team project
			return nil, fmt.Errorf("GCP resources requested, but no team project ID annotation set on namespace %s (not running on GCP?)", app.GetNamespace())
		}
		rollout.ResourceOptions.GoogleTeamProjectId = projectID
	}

	// Create Linkerd resources only if feature is enabled and namespace is Linkerd-enabled
	if n.Config.Features.Linkerd && namespace.Annotations["linkerd.io/inject"] == "enabled" {
		rollout.ResourceOptions.Linkerd = true
	}

	if rollout.ResourceOptions.DigdiratorEnabled && app.Spec.IDPorten != nil && app.Spec.IDPorten.Enabled && app.Spec.IDPorten.Sidecar != nil && app.Spec.IDPorten.Sidecar.Enabled {
		rollout.ResourceOptions.WonderwallEnabled = true
	}

	rollout.SetCurrentDeployment(previousDeployment, *app.Spec.Replicas.Min)
	rollout.ResourceOperations, err = resourcecreator.CreateApplication(app, rollout.ResourceOptions)

	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}

// ClusterOperations generates a set of functions that will perform the rollout in the cluster.
func (n *Synchronizer) ClusterOperations(ctx context.Context, rollout Rollout) []commit {
	funcs := make([]commit, 0)
	deletes := make([]commit, 0)

	// A wrapper to get GroupVersionKind but ensure there's no nils.
	getGroupVersionKind := func(o runtime.Object) schema.GroupVersionKind {
		if o == nil || o.GetObjectKind() == nil {
			return schema.GroupVersionKind{}
		}
		return o.GetObjectKind().GroupVersionKind()
	}

	for _, rop := range rollout.ResourceOperations {
		c := commit{
			groupVersionKind: getGroupVersionKind(rop.Resource),
		}
		switch rop.Operation {
		case resource.OperationCreateOrUpdate:
			c.fn = updater.CreateOrUpdate(ctx, n, n.Scheme, rop.Resource)
		case resource.OperationCreateOrRecreate:
			c.fn = updater.CreateOrRecreate(ctx, n, rop.Resource)
		case resource.OperationCreateIfNotExists:
			c.fn = updater.CreateIfNotExists(ctx, n, rop.Resource)
		case resource.OperationDeleteIfExists:
			c.fn = updater.DeleteIfExists(ctx, n, rop.Resource)
		default:
			return []commit{
				{
					fn: func() error {
						return fmt.Errorf("BUG: no such operation %s", rop.Operation)
					},
				},
			}
		}

		funcs = append(funcs, c)
	}

	// Delete extraneous resources
	unreferenced, err := n.Unreferenced(ctx, rollout)
	if err != nil {
		deletes = append(deletes, commit{fn: func() error {
			return fmt.Errorf("unable to clean up obsolete resources: %s", err)
		}})
	} else {
		for _, rsrc := range unreferenced {
			deletes = append(deletes, commit{
				groupVersionKind: getGroupVersionKind(rsrc),
				fn:               updater.DeleteIfExists(ctx, n, rsrc),
			})
		}
	}

	return append(deletes, funcs...)
}

var appsync sync.Mutex

// UpdateApplication atomically update an Application resource.
// Locks the resource to avoid race conditions.
func (n *Synchronizer) UpdateApplication(ctx context.Context, source resource.Source, updateFunc func(existing *nais_io_v1alpha1.Application) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	existing := &nais_io_v1alpha1.Application{}
	err := n.Get(ctx, client.ObjectKey{Namespace: source.GetNamespace(), Name: source.GetName()}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of Application: %s", err)
	}

	return updateFunc(existing)
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
