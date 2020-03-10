package synchronizer

import (
	"fmt"
	"sync"
	"time"

	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/updater"
	log "github.com/sirupsen/logrus"
	istioClient "istio.io/client-go/pkg/clientset/versioned"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	workQueue       chan v1alpha1.Application
	ClientSet       kubernetes.Interface
	AppClient       clientV1Alpha1.Interface
	IstioClient     istioClient.Interface
	ResourceOptions resourcecreator.ResourceOptions
	Config          Config
}

type Config struct {
	KafkaEnabled               bool
	DeploymentMonitorFrequency time.Duration
	DeploymentMonitorTimeout   time.Duration
}

func New(clientSet kubernetes.Interface, appClient clientV1Alpha1.Interface, istioClient istioClient.Interface, resourceOptions resourcecreator.ResourceOptions, config Config) *Synchronizer {
	naiserator := Synchronizer{
		workQueue:       make(chan v1alpha1.Application, 1024),
		ClientSet:       clientSet,
		AppClient:       appClient,
		IstioClient:     istioClient,
		Config:          config,
		ResourceOptions: resourceOptions,
	}

	return &naiserator
}

// Creates a Kubernetes event.
func (n *Synchronizer) reportEvent(reportedEvent *corev1.Event) (*corev1.Event, error) {
	events, err := n.ClientSet.CoreV1().Events(reportedEvent.Namespace).List(v1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.uid=%s", reportedEvent.InvolvedObject.Name, reportedEvent.InvolvedObject.UID),
	})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("while getting events for app %s, got error: %s", reportedEvent.InvolvedObject.Name, err)
	}

	for _, event := range events.Items {
		if event.Message == reportedEvent.Message {
			event.Count++
			event.LastTimestamp = reportedEvent.LastTimestamp
			return n.ClientSet.CoreV1().Events(event.Namespace).Update(&event)
		}
	}

	return n.ClientSet.CoreV1().Events(reportedEvent.Namespace).Create(reportedEvent)
}

// Reports an error through the error log, a Kubernetes event, and possibly logs a failure in event creation.
func (n *Synchronizer) reportError(source string, err error, app *v1alpha1.Application) {
	logger := log.WithFields(app.LogFields())
	logger.Error(err)
	_, err = n.reportEvent(app.CreateEvent(source, err.Error(), "Warning"))
	if err != nil {
		logger.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

// Process work queue
func (n *Synchronizer) Process(app *v1alpha1.Application) {
	changed := true

	logger := *log.WithFields(app.LogFields())

	// Update Application resource with deployment event
	defer func() {
		if !changed {
			return
		}
		err := n.UpdateApplication(app, func(existing *v1alpha1.Application) error {
			existing.Status = app.Status
			_, err := n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(existing)
			return err
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
		return
	}

	if rollout == nil {
		changed = false
		logger.Debugf("No changes")
		return
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
		return
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
	app.SetDeploymentRolloutStatus(event.RolloutStatus)

	if n.Config.KafkaEnabled && !app.SkipDeploymentMessage() {
		kafka.Events <- kafka.Message{Event: event, Logger: logger}
	}

	// Monitor the rollout status so that we can report a successfully completed rollout to NAIS deploy.
	go n.MonitorRollout(*app, logger, n.Config.DeploymentMonitorFrequency, n.Config.DeploymentMonitorTimeout)
}

// Process work queue
func (n *Synchronizer) Main() {
	log.Info("Main loop consuming from work queue")

	for app := range n.workQueue {
		metrics.QueueSize.Set(float64(len(n.workQueue)))
		n.Process(&app)
	}
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
func (n *Synchronizer) Prepare(app *v1alpha1.Application) (*Rollout, error) {
	var err error

	rollout := &Rollout{
		App:             app,
		ResourceOptions: n.ResourceOptions,
	}

	if err = v1alpha1.ApplyDefaults(app); err != nil {
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
	previousDeployment, err := n.ClientSet.AppsV1().Deployments(app.Namespace).Get(app.Name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing deployment: %s", err)
	}

	if app.Spec.GCP != nil && app.Spec.GCP.SqlInstances != nil {
		namespace, err := n.ClientSet.CoreV1().Namespaces().Get(app.GetNamespace(), v1.GetOptions{})

		if err != nil && !errors.IsNotFound(err) {
			return nil, fmt.Errorf("query existing namespace: %s", err)
		}

		if val, ok := namespace.Annotations["cnrm.cloud.google.com/project-id"]; ok {
			rollout.SetGoogleTeamProjectId(val)
		} else {
			return nil, fmt.Errorf("team project id annotation not set on namespace %s", app.GetNamespace())
		}
	}

	rollout.SetCurrentDeployment(previousDeployment)
	rollout.ResourceOperations, err = resourcecreator.Create(app, rollout.ResourceOptions)
	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}

func (n *Synchronizer) Enqueue(app *v1alpha1.Application) {
	// Create a copy of the application with defaults applied.
	// This copy is used as a basis for creating resources.
	// The original application resource must be persisted back with the rollout status,
	// and preserving it in its original state without defaults is preferred.
	n.workQueue <- *app.DeepCopy()
}

// ClusterOperations generates a set of functions that will perform the rollout in the cluster.
func (n *Synchronizer) ClusterOperations(rollout Rollout) []func() error {
	var fn func() error

	funcs := make([]func() error, 0)

	for _, rop := range rollout.ResourceOperations {
		switch rop.Operation {
		case resourcecreator.OperationCreateOrUpdate:
			fn = updater.CreateOrUpdate(n.ClientSet, n.AppClient, n.IstioClient, rop.Resource)
		case resourcecreator.OperationCreateOrRecreate:
			fn = updater.CreateOrRecreate(n.ClientSet, n.AppClient, n.IstioClient, rop.Resource)
		case resourcecreator.OperationCreateIfNotExists:
			fn = updater.CreateIfNotExists(n.ClientSet, n.AppClient, n.IstioClient, rop.Resource)
		case resourcecreator.OperationDeleteIfExists:
			fn = updater.DeleteIfExists(n.ClientSet, n.AppClient, n.IstioClient, rop.Resource)
		default:
			log.Fatalf("BUG: no such operation %s", rop.Operation)
		}

		funcs = append(funcs, fn)
	}

	return funcs
}

var appsync sync.Mutex

// Atomically update an Application resource.
// Locks the resource to avoid race conditions.
func (n *Synchronizer) UpdateApplication(app *v1alpha1.Application, updateFunc func(existing *v1alpha1.Application) error) error {
	var err error

	appsync.Lock()
	defer appsync.Unlock()

	app, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Get(app.Name, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get newest version of Application: %s", err)
	}

	return updateFunc(app)
}

func (n *Synchronizer) MonitorRollout(app v1alpha1.Application, logger log.Entry, frequency, timeout time.Duration) {
	logger.Debugf("monitoring rollout status")

	for {
		select {
		case <-time.After(frequency):
			deploy, err := n.ClientSet.AppsV1().Deployments(app.Namespace).Get(app.Name, v1.GetOptions{})
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
				err = n.UpdateApplication(&app, func(app *v1alpha1.Application) error {
					app.Status.SynchronizationState = EventRolloutComplete
					app.Status.RolloutCompleteTime = time.Now().UnixNano()
					app.SetDeploymentRolloutStatus(event.RolloutStatus)
					_, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(app)
					return err
				})

				if err != nil {
					logger.Errorf("monitor rollout: store application sync status: %s", err)
				}

				return
			}
		case <-time.After(timeout):
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
