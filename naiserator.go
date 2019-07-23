package naiserator

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	DeploymentMonitorFrequency = time.Second * 5
	DeploymentMonitorTimeout   = time.Minute * 5
)

// Naiserator is a singleton that holds Kubernetes client instances.
type Naiserator struct {
	ClientSet                 kubernetes.Interface
	KafkaEnabled              bool
	AppClient                 *clientV1Alpha1.Clientset
	ApplicationInformer       informers.ApplicationInformer
	ApplicationInformerSynced cache.InformerSynced
	ResourceOptions           resourcecreator.ResourceOptions
}

func NewNaiserator(clientSet kubernetes.Interface, appClient *clientV1Alpha1.Clientset, applicationInformer informers.ApplicationInformer, resourceOptions resourcecreator.ResourceOptions, kafkaEnabled bool) *Naiserator {
	naiserator := Naiserator{
		ClientSet:                 clientSet,
		AppClient:                 appClient,
		ApplicationInformer:       applicationInformer,
		ApplicationInformerSynced: applicationInformer.Informer().HasSynced,
		KafkaEnabled:              kafkaEnabled,
		ResourceOptions:           resourceOptions}

	applicationInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(newPod interface{}) {
				naiserator.add(newPod)
			},
			UpdateFunc: func(oldPod, newPod interface{}) {
				naiserator.update(oldPod, newPod)
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
	log.Error(err)
	_, err = n.reportEvent(app.CreateEvent(source, err.Error(), "Warning"))
	if err != nil {
		log.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

func (n *Naiserator) synchronize(logger *log.Entry, app *v1alpha1.Application) error {
	if err := v1alpha1.ApplyDefaults(app); err != nil {
		return fmt.Errorf("while applying default values to application spec: %s", err)
	}

	hash, err := app.Hash()
	if err != nil {
		return fmt.Errorf("while hashing application spec: %s", err)
	}
	if app.LastSyncedHash() == hash {
		logger.Infof("%s: no changes", app.Name)
		return nil
	}

	deploymentID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("while generating a deployment UUID: %s", err)
	}
	app.SetCorrelationID(deploymentID.String())
	logger = logger.WithField("correlation-id", deploymentID.String())

	if err := n.removeOrphanIngresses(logger, app); err != nil {
		return fmt.Errorf("while removing old resources: %s", err)
	}

	// If the autoscaler is unavailable when a deployment is made, we risk scaling the application to the default
	// number of replicas, which is set to one by default. To avoid this, we need to check the existing deployment
	// resource and pass the correct number in the resource options.
	//
	// The number of replicas is set to whichever is highest: the current number of replicas (which might be zero),
	// or the default number of replicas.
	deploymentResource, err := n.ClientSet.AppsV1().Deployments(app.Namespace).Get(app.Name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("while querying existing deployment: %s", err)
	} else if deploymentResource != nil && deploymentResource.Spec.Replicas != nil {
		n.ResourceOptions.NumReplicas = max(1, *deploymentResource.Spec.Replicas)
	} else {
		n.ResourceOptions.NumReplicas = max(1, int32(app.Spec.Replicas.Min))
	}

	resources, err := resourcecreator.Create(app, n.ResourceOptions)
	if err != nil {
		return fmt.Errorf("while creating resources: %s", err)
	}

	if err := n.createOrUpdateMany(resources); err != nil {
		return fmt.Errorf("while persisting resources to Kubernetes: %s", err)
	}

	// At this point, the deployment is complete. All that is left is to register the application hash and cache it,
	// so that the deployment does not happen again. Thus, we update the metrics before the end of the function.
	metrics.ResourcesGenerated.Add(float64(len(resources)))
	metrics.Deployments.Inc()

	if n.KafkaEnabled {
		// Broadcast a message on Kafka that the deployment is initialized.
		event := generator.NewDeploymentEvent(*app)
		kafka.Events <- kafka.Message{Event: event, Logger: *logger}

		app.SetDeploymentRolloutStatus(event.RolloutStatus)

		// Monitor its completion timeline over a designated period
		go n.MonitorRollout(*app, *logger, DeploymentMonitorFrequency, DeploymentMonitorTimeout)
	}

	app.SetLastSyncedHash(hash)
	logger.Infof("%s: setting new hash %s", app.Name, hash)

	app.NilFix()
	_, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(app)
	if err != nil {
		return fmt.Errorf("while storing application sync metadata: %s", err)
	}

	_, err = n.reportEvent(app.CreateEvent("synchronize", fmt.Sprintf("successfully synchronized application resources (hash = %s)", hash), "Normal"))
	if err != nil {
		logger.Errorf("While creating an event for this error, another error occurred: %s", err)
	}

	return nil
}

func (n *Naiserator) update(old, new interface{}) {
	var app *v1alpha1.Application
	if new != nil {
		app = new.(*v1alpha1.Application)
	}

	logger := log.WithFields(log.Fields{
		"namespace":       app.Namespace,
		"apiversion":      app.APIVersion,
		"resourceversion": app.ResourceVersion,
		"application":     app.Name,
	})

	metrics.ApplicationsProcessed.Inc()
	logger.Infof("%s: synchronizing application", app.Name)

	if err := n.synchronize(logger, app); err != nil {
		metrics.ApplicationsFailed.Inc()
		logger.Errorf("%s: error %s", app.Name, err)
		n.reportError("synchronize", err, app)
	} else {
		logger.Infof("%s: synchronized successfully", app.Name)
	}

	logger.Infof("%s: finished synchronizing", app.Name)
}

func (n *Naiserator) add(app interface{}) {
	n.update(nil, app)
}

func (n *Naiserator) createOrUpdateMany(resources []runtime.Object) error {
	var result = &multierror.Error{}

	for _, resource := range resources {
		err := updater.Updater(n.ClientSet, n.AppClient, resource)()
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func (n *Naiserator) removeOrphanIngresses(logger *log.Entry, app *v1alpha1.Application) error {
	if len(app.Spec.Ingresses) > 0 {
		return nil
	}

	err := n.ClientSet.ExtensionsV1beta1().Ingresses(app.Namespace).Delete(app.Name, &v1.DeleteOptions{})

	if errors.IsNotFound(err) {
		return nil
	}

	logger.Infof("%s: successfully deleted orphan ingresses", app.Name)

	return err
}

func (n *Naiserator) MonitorRollout(app v1alpha1.Application, logger log.Entry, frequency, timeout time.Duration) {
	for {
		select {
		case <-time.After(frequency):
			deploy, err := n.ClientSet.AppsV1().Deployments(app.Namespace).Get(app.Name, v1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("%s: While trying to get Deployment for app: %s", app.Name, err)
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
					logger.Errorf("while trying to get newest version of Application %s: %s", app.Name, err)
					return
				}

				updatedApp.SetDeploymentRolloutStatus(event.RolloutStatus)
				_, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(updatedApp)
				if err != nil {
					logger.Errorf("while storing application sync status: %s", err)
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
