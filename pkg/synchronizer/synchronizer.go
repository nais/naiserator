package synchronizer

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	informers "github.com/nais/naiserator/pkg/client/informers/externalversions/nais.io/v1alpha1"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	DeploymentMonitorFrequency = time.Second * 5
	DeploymentMonitorTimeout   = time.Minute * 5
)

// Naiserator is a singleton that holds Kubernetes client instances.
type Naiserator struct {
	workQueue                 chan v1alpha1.Application
	ClientSet                 kubernetes.Interface
	KafkaEnabled              bool
	AppClient                 *clientV1Alpha1.Clientset
	ApplicationInformer       informers.ApplicationInformer
	ApplicationInformerSynced cache.InformerSynced
	ResourceOptions           resourcecreator.ResourceOptions
}

func New(clientSet kubernetes.Interface, appClient *clientV1Alpha1.Clientset, applicationInformer informers.ApplicationInformer, resourceOptions resourcecreator.ResourceOptions, kafkaEnabled bool) *Naiserator {
	naiserator := Naiserator{
		workQueue:                 make(chan v1alpha1.Application, 1024),
		ClientSet:                 clientSet,
		AppClient:                 appClient,
		ApplicationInformer:       applicationInformer,
		ApplicationInformerSynced: applicationInformer.Informer().HasSynced,
		KafkaEnabled:              kafkaEnabled,
		ResourceOptions:           resourceOptions}

	applicationInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(newPod interface{}) {
				naiserator.watchCallback(newPod)
			},
			UpdateFunc: func(oldPod, newPod interface{}) {
				naiserator.watchCallback(newPod)
			},
		})

	return &naiserator
}

// Creates a Kubernetes event.
func (n *Naiserator) reportEvent(reportedEvent *corev1.Event) (*corev1.Event, error) {
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
func (n *Naiserator) reportError(source string, err error, app *v1alpha1.Application) {
	logger := log.WithFields(app.LogFields())
	logger.Error(err)
	_, err = n.reportEvent(app.CreateEvent(source, err.Error(), "Warning"))
	if err != nil {
		logger.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

// Process work queue
func (n *Naiserator) Process(app v1alpha1.Application) {
	rollout, err := n.Prepare(app)
	if err != nil {
		n.reportError("prepare", err, &app)
		return
	}

	logger := *log.WithFields(app.LogFields())

	if rollout == nil {
		logger.Debugf("no changes")
		return
	}

	logger = *log.WithFields(rollout.App.LogFields())
	logger.Debugf("starting synchronization")

	err = n.Sync(*rollout)
	if err != nil {
		n.reportError("synchronize", err, &app)
		return
	}

	// Synchronization OK
	logger.Debugf("successful synchronization")
	metrics.ApplicationsProcessed.Inc()
	metrics.Deployments.Inc()

	_, err = n.reportEvent(app.CreateEvent("synchronize", "successfully synchronized application resources", "Normal"))
	if err != nil {
		log.Errorf("While creating an event for this rollout, an error occurred: %s", err)
	}

	// Create new deployment event
	event := generator.NewDeploymentEvent(app)
	app.SetDeploymentRolloutStatus(event.RolloutStatus)

	// Update Application resource with deployment event
	err = n.UpdateStatus(app, *rollout)
	if err != nil {
		n.reportError("main", err, &app)
	} else {
		logger.Debugf("persisted Application status")
	}

	// If Kafka is enabled, we can send out a signal when the deployment is complete.
	// To do this we need to monitor the rollout status over a designated period.
	if n.KafkaEnabled {
		kafka.Events <- kafka.Message{Event: event, Logger: logger}
		go n.MonitorRollout(app, logger, DeploymentMonitorFrequency, DeploymentMonitorTimeout)
	}
}

// Process work queue
func (n *Naiserator) Main() {
	log.Info("Main loop consuming from work queue")

	for app := range n.workQueue {
		n.Process(app)
	}
}

// Update application status and persist to cluster
func (n *Naiserator) UpdateStatus(app v1alpha1.Application, rollout Rollout) error {
	app.SetCorrelationID(rollout.App.Status.CorrelationID)
	app.SetLastSyncedHash(rollout.App.LastSyncedHash())
	// app.NilFix()  // FIXME unneeded?????

	_, err := n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(&app)
	if err != nil {
		return fmt.Errorf("persist application: %s", err)
	}

	return nil
}

func (n *Naiserator) Sync(rollout Rollout) error {

	commits := n.ClusterOperations(rollout)

	for _, fn := range commits {
		if err := fn(); err != nil {
			metrics.ApplicationsFailed.Inc()
			return fmt.Errorf("persisting resource to Kubernetes: %s", err)
		}
		metrics.ResourcesGenerated.Inc()
	}

	return nil
}

// Prepare converts a NAIS application spec into a Rollout object.
// This is a read-only operation
// The Rollout object contains callback functions that commits changes in the cluster.
func (n *Naiserator) Prepare(app v1alpha1.Application) (*Rollout, error) {
	if err := v1alpha1.ApplyDefaults(&app); err != nil {
		return nil, fmt.Errorf("BUG: merge default values into application: %s", err)
	}

	hash, err := app.Hash()
	if err != nil {
		return nil, fmt.Errorf("BUG: create application hash: %s", err)
	}

	// Skip processing if application didn't change since last synchronization.
	if app.LastSyncedHash() == hash {
		return nil, nil
	}
	app.SetLastSyncedHash(hash)

	deploymentID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("BUG: generate deployment correlation ID: %s", err)
	}
	app.SetCorrelationID(deploymentID.String())

	// Make a query to Kubernetes for this application's previous deployment.
	// The number of replicas is significant, so we need to carry it over to match
	// this next rollout.
	previousDeployment, err := n.ClientSet.AppsV1().Deployments(app.Namespace).Get(app.Name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing deployment: %s", err)
	}

	rollout := &Rollout{
		App:             &app,
		ResourceOptions: n.ResourceOptions,
	}
	rollout.SetCurrentDeployment(previousDeployment)
	rollout.ResourceOperations, err = resourcecreator.Create(&app, rollout.ResourceOptions)
	if err != nil {
		return nil, fmt.Errorf("creating cluster resource operations: %s", err)
	}

	return rollout, nil
}

func (n *Naiserator) Enqueue(app *v1alpha1.Application) {
	// Create a copy of the application with defaults applied.
	// This copy is used as a basis for creating resources.
	// The original application resource must be persisted back with the rollout status,
	// and preserving it in its original state without defaults is preferred.
	n.workQueue <- *app.DeepCopy()
}

// This function is passed to the Application resource watcher.
// It will be called once for every application resource, which
// must be type cast from interface{} to Application.
func (n *Naiserator) watchCallback(unstructured interface{}) {
	var app *v1alpha1.Application
	var ok bool

	if unstructured == nil {
		return
	}

	app, ok = unstructured.(*v1alpha1.Application)
	if !ok {
		// type cast failed; discard
		log.Errorf("watchCallback encountered invalid Application resource of type %T", unstructured)
		return
	}

	n.Enqueue(app)
}

// ClusterOperations generates a set of functions that will perform the rollout in the cluster.
func (n *Naiserator) ClusterOperations(rollout Rollout) []func() error {
	var fn func() error

	funcs := make([]func() error, 0)

	for _, rop := range rollout.ResourceOperations {
		switch rop.Operation {
		case resourcecreator.OperationCreateOrUpdate:
			fn = updater.CreateOrUpdate(n.ClientSet, n.AppClient, rop.Resource)
		case resourcecreator.OperationCreateOrRecreate:
			fn = updater.CreateOrRecreate(n.ClientSet, n.AppClient, rop.Resource)
		case resourcecreator.OperationDeleteIfExists:
			fn = updater.DeleteIfExists(n.ClientSet, n.AppClient, rop.Resource)
		}

		funcs = append(funcs, fn)
	}

	return funcs
}

func (n *Naiserator) MonitorRollout(app v1alpha1.Application, logger log.Entry, frequency, timeout time.Duration) {
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
				kafka.Events <- kafka.Message{Event: event, Logger: logger}

				// During this time the app has been updated, so we need to acquire the newest version before proceeding
				var updatedApp *v1alpha1.Application
				updatedApp, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Get(app.Name, v1.GetOptions{})
				if err != nil {
					logger.Errorf("monitor rollout: get newest version of Application: %s", err)
					return
				}

				updatedApp.SetDeploymentRolloutStatus(event.RolloutStatus)
				_, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(updatedApp)
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

func (n *Naiserator) Run(stop <-chan struct{}) {
	log.Info("Starting application synchronization")
	if !cache.WaitForCacheSync(stop, n.ApplicationInformerSynced) {
		log.Error("timed out waiting for cache sync")
		return
	}
}

func max(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
