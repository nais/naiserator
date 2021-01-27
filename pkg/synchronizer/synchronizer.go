package synchronizer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/updater"
	log "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// Synchronizer creates child resources from Application resources in the cluster.
// If the child resources does not match the Application spec, the resources are updated.
type Synchronizer struct {
	client.Client
	ResourceOptions resourcecreator.ResourceOptions
	Config          Config
}

type Config struct {
	KafkaEnabled               bool
	QueueSize                  int
	DeploymentMonitorFrequency time.Duration
	DeploymentMonitorTimeout   time.Duration
}

func (n *Synchronizer) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1alpha1.Application{}).
		Complete(n)
}

// Creates a Kubernetes event, or updates an existing one with an incremented counter
func (n *Synchronizer) reportEvent(reportedEvent *corev1.Event) (*corev1.Event, error) {
	selector, err := fields.ParseSelector(fmt.Sprintf("involvedObject.name=%s,involvedObject.uid=%s", reportedEvent.InvolvedObject.Name, reportedEvent.InvolvedObject.UID))
	if err != nil {
		return nil, err // fixme
	}
	ctx := context.Background() // fixme
	events := &corev1.EventList{}
	err = n.Client.List(ctx, events, &client.ListOptions{
		FieldSelector: selector,
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("while getting events for app %s, got error: %s", reportedEvent.InvolvedObject.Name, err)
	}

	for _, event := range events.Items {
		if event.Message == reportedEvent.Message {
			event.Count++
			event.LastTimestamp = reportedEvent.LastTimestamp
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
func (n *Synchronizer) reportError(source string, err error, app *nais_io_v1alpha1.Application) {
	logger := log.WithFields(app.LogFields())
	logger.Error(err)
	_, err = n.reportEvent(app.CreateEvent(source, err.Error(), "Warning"))
	if err != nil {
		logger.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

// Process work queue
func (n *Synchronizer) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	app := &nais_io_v1alpha1.Application{}
	ctx := context.Background() // fixme
	err := n.Get(ctx, req.NamespacedName, app)
	if err != nil {
		panic("errors not handled; delete not implemented")
	}

	changed := true

	logger := *log.WithFields(app.LogFields())

	// Update Application resource with deployment event
	defer func() {
		if !changed {
			return
		}
		err := n.UpdateApplication(app, func(existing *nais_io_v1alpha1.Application) error {
			existing.Status = app.Status
			return n.Update(ctx, app)
		})
		if err != nil {
			n.reportError(EventFailedStatusUpdate, err, app)
		} else {
			logger.Debugf("Application status: %+v'", app.Status)
		}
	}()

	rollout, err := n.Prepare(app)
	if err != nil {
		app.Status.SynchronizationState = EventFailedPrepare
		n.reportError(app.Status.SynchronizationState, err, app)
		return ctrl.Result{}, err // fixme
	}

	if rollout == nil {
		changed = false
		logger.Debugf("No changes")
		return ctrl.Result{}, nil
	}

	logger = *log.WithFields(rollout.App.LogFields())
	logger.Debugf("Starting synchronization")
	metrics.ApplicationsProcessed.Inc()

	app.Status.CorrelationID = rollout.CorrelationID

	err, retry := n.Sync(*rollout)
	if err != nil {
		if retry {
			app.Status.SynchronizationState = EventRetrying
			metrics.Retries.Inc()
		} else {
			app.Status.SynchronizationState = EventFailedSynchronization
			app.Status.SynchronizationHash = rollout.SynchronizationHash // permanent failure
			metrics.ApplicationsFailed.Inc()
		}
		n.reportError(app.Status.SynchronizationState, err, app)
		return ctrl.Result{}, nil // fixme?
	}

	// Synchronization OK
	logger.Debugf("Successful synchronization")
	app.Status.SynchronizationState = EventSynchronized
	app.Status.SynchronizationHash = rollout.SynchronizationHash
	app.Status.SynchronizationTime = time.Now().UnixNano()
	metrics.Deployments.Inc()

	_, err = n.reportEvent(app.CreateEvent(app.Status.SynchronizationState, "Successfully synchronized all application resources", "Normal"))
	if err != nil {
		log.Errorf("While creating an event for this rollout, an error occurred: %s", err)
	}

	// Create new deployment event
	event := generator.NewDeploymentEvent(*app)
	app.SetDeploymentRolloutStatus(event.RolloutStatus.String())

	if n.Config.KafkaEnabled && !app.SkipDeploymentMessage() {
		kafka.Events <- kafka.Message{Event: event, Logger: logger}
	}

	// Monitor the rollout status so that we can report a successfully completed rollout to NAIS deploy.
	go n.MonitorRollout(*app, logger, n.Config.DeploymentMonitorFrequency, n.Config.DeploymentMonitorTimeout)

	return ctrl.Result{}, nil
}

// Return all resources in cluster which was created by synchronizer previously, but is not included in the current rollout.
func (n *Synchronizer) Unreferenced(rollout Rollout) ([]runtime.Object, error) {
	ctx := context.Background() // fixme

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

	resources, err := updater.FindAll(ctx, n, rollout.App)
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

func (n *Synchronizer) Sync(rollout Rollout) (error, bool) {

	commits := n.ClusterOperations(rollout)

	for _, fn := range commits {
		if err := observeDuration(fn); err != nil {
			retry := false
			// In case of race condition errors
			if errors.IsConflict(err) {
				retry = true
			}
			reason := errors.ReasonForError(err)
			return fmt.Errorf("persisting resource to Kubernetes: %s: %s", reason, err), retry
		}
		metrics.ResourcesGenerated.Inc()
	}

	return nil, false
}

// Prepare converts a NAIS application spec into a Rollout object.
// This is a read-only operation
// The Rollout object contains callback functions that commits changes in the cluster.
func (n *Synchronizer) Prepare(app *nais_io_v1alpha1.Application) (*Rollout, error) {
	ctx := context.Background()
	var err error

	rollout := &Rollout{
		App:             app,
		ResourceOptions: n.ResourceOptions,
	}

	if err = nais_io_v1alpha1.ApplyDefaults(app); err != nil {
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

	rollout.CorrelationID, err = app.NextCorrelationID()
	if err != nil {
		return nil, err
	}

	// Make a query to Kubernetes for this application's previous deployment.
	// The number of replicas is significant, so we need to carry it over to match
	// this next rollout.
	previousDeployment := &apps.Deployment{}
	err = n.Get(ctx, client.ObjectKey{Name: app.GetName(), Namespace: app.GetNamespace()}, previousDeployment)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing deployment: %s", err)
	}

	if app.Spec.GCP != nil && (app.Spec.GCP.SqlInstances != nil || app.Spec.GCP.Permissions != nil) {
		namespace := &corev1.Namespace{}
		err = n.Get(ctx, client.ObjectKey{Namespace: app.GetNamespace()}, namespace)

		if err != nil && !errors.IsNotFound(err) {
			return nil, fmt.Errorf("query existing namespace: %s", err)
		}

		if val, ok := namespace.Annotations["cnrm.cloud.google.com/project-id"]; ok {
			rollout.SetGoogleTeamProjectId(val)
		} else {
			return nil, fmt.Errorf("GCP resources requested, but no team project ID annotation set on namespace %s (not running on GCP?)", app.GetNamespace())
		}
	}

	rollout.SetCurrentDeployment(previousDeployment)
	rollout.ResourceOperations, err = resourcecreator.Create(app, rollout.ResourceOptions)
	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}

// ClusterOperations generates a set of functions that will perform the rollout in the cluster.
func (n *Synchronizer) ClusterOperations(rollout Rollout) []func() error {
	ctx := context.Background() // fixme
	var fn func() error

	funcs := make([]func() error, 0)

	for _, rop := range rollout.ResourceOperations {
		switch rop.Operation {
		case resourcecreator.OperationCreateOrUpdate:
			fn = updater.CreateOrUpdate(ctx, n, rop.Resource)
		case resourcecreator.OperationCreateOrRecreate:
			fn = updater.CreateOrRecreate(ctx, n, rop.Resource)
		case resourcecreator.OperationCreateIfNotExists:
			fn = updater.CreateIfNotExists(ctx, n, rop.Resource)
		default:
			log.Fatalf("BUG: no such operation %s", rop.Operation)
		}

		funcs = append(funcs, fn)
	}

	// Delete extraneous resources
	unreferenced, err := n.Unreferenced(rollout)
	if err != nil {
		funcs = append(funcs, func() error {
			return fmt.Errorf("unable to clean up obsolete resources: %s", err)
		})
	} else {
		for _, resource := range unreferenced {
			funcs = append(funcs, updater.DeleteIfExists(ctx, n, resource))
		}
	}

	return funcs
}

var appsync sync.Mutex

// Atomically update an Application resource.
// Locks the resource to avoid race conditions.
func (n *Synchronizer) UpdateApplication(app *nais_io_v1alpha1.Application, updateFunc func(existing *nais_io_v1alpha1.Application) error) error {
	appsync.Lock()
	defer appsync.Unlock()

	ctx := context.Background() // fixme
	newapp := &nais_io_v1alpha1.Application{}
	err := n.Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, app)
	if err != nil {
		return fmt.Errorf("get newest version of Application: %s", err)
	}

	return updateFunc(newapp)
}

func (n *Synchronizer) MonitorRollout(app nais_io_v1alpha1.Application, logger log.Entry, frequency, timeout time.Duration) {
	logger.Debugf("monitoring rollout status")
	ctx := context.Background() // fixme

	for {
		select {
		case <-time.After(frequency):
			deploy := &apps.Deployment{}
			err := n.Get(ctx, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, deploy)
			if err != nil {
				if !errors.IsNotFound(err) {
					logger.Errorf("monitor rollout: failed to query Deployment: %s", err)
				}
				continue
			}

			if deploymentComplete(deploy, &deploy.Status) {
				event := generator.NewDeploymentEvent(app)
				event.RolloutStatus = deployment.RolloutStatus_complete
				if n.Config.KafkaEnabled && !app.SkipDeploymentMessage() {
					kafka.Events <- kafka.Message{Event: event, Logger: logger}
				}

				_, err = n.reportEvent(app.CreateEvent(EventRolloutComplete, "Deployment rollout has completed", "Normal"))
				if err != nil {
					logger.Errorf("monitor rollout: unable to report rollout complete event: %s", err)
				}

				// During this time the app has been updated, so we need to acquire the newest version before proceeding
				err = n.UpdateApplication(&app, func(app *nais_io_v1alpha1.Application) error {
					app.Status.SynchronizationState = EventRolloutComplete
					app.Status.RolloutCompleteTime = time.Now().UnixNano()
					app.SetDeploymentRolloutStatus(event.RolloutStatus.String())
					return n.Update(ctx, app)
				})

				if err != nil {
					logger.Errorf("monitor rollout: store application sync status: %s", err)
				}

				return
			}
		case <-time.After(timeout):
			logger.Debugf("application has not rolled out completely in %s; giving up", timeout.String())
			return
		}
	}
}

// deploymentComplete considers a deployment to be complete once all of its desired replicas
// are updated and available, and no old pods are running.
//
// Copied verbatim from
// https://github.com/kubernetes/kubernetes/blob/74bcefc8b2bf88a2f5816336999b524cc48cf6c0/pkg/controller/deployment/util/deployment_util.go#L745
func deploymentComplete(deployment *apps.Deployment, newStatus *apps.DeploymentStatus) bool {
	return newStatus.UpdatedReplicas == *(deployment.Spec.Replicas) &&
		newStatus.Replicas == *(deployment.Spec.Replicas) &&
		newStatus.AvailableReplicas == *(deployment.Spec.Replicas) &&
		newStatus.ObservedGeneration >= deployment.Generation
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
