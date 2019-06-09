package naiserator

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	informers "github.com/nais/naiserator/pkg/client/informers/externalversions/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/updater"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Naiserator is a singleton that holds Kubernetes client instances.
type Naiserator struct {
	ClientSet                 kubernetes.Interface
	AppClient                 *clientV1Alpha1.Clientset
	ApplicationInformer       informers.ApplicationInformer
	ApplicationInformerSynced cache.InformerSynced
	enableAccessPolicy        bool
}

func NewNaiserator(clientSet kubernetes.Interface, appClient *clientV1Alpha1.Clientset, applicationInformer informers.ApplicationInformer, enableAccessPolicy bool) *Naiserator {
	naiserator := Naiserator{
		ClientSet:                 clientSet,
		AppClient:                 appClient,
		ApplicationInformer:       applicationInformer,
		ApplicationInformerSynced: applicationInformer.Informer().HasSynced,
		enableAccessPolicy:        enableAccessPolicy}

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
func (n *Naiserator) reportEvent(event *corev1.Event) (*corev1.Event, error) {
	return n.ClientSet.CoreV1().Events(event.Namespace).Create(event)
}

// Reports an error through the error log, a Kubernetes event, and possibly logs a failure in event creation.
func (n *Naiserator) reportError(source string, err error, app *v1alpha1.Application) {
	log.Error(err)
	ev := app.CreateEvent(source, err.Error(), "Warning")
	_, err = n.reportEvent(ev)
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

	// If the autoscaler is unavailable when a deployment is made, we risk scaling the application to the default
	// number of replicas, which is set to one by default. To avoid this, we need to check the existing deployment
	// resource and pass the correct number in the resource options.
	resourceOptions := resourcecreator.NewResourceOptions()
	deployment, err := n.ClientSet.AppsV1().Deployments(app.Namespace).Get(app.Name, v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("while querying existing deployment: %s", err)
	} else if deployment != nil && deployment.Spec.Replicas != nil {
		resourceOptions.NumReplicas = *deployment.Spec.Replicas
	}

	// AccessPolicy ensures that Istio resources are created.
	resourceOptions.AccessPolicy = n.enableAccessPolicy

	resources, err := resourcecreator.Create(app, resourceOptions)
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

func (n *Naiserator) Run(stop <-chan struct{}) {
	log.Info("Starting application synchronization")
	if !cache.WaitForCacheSync(stop, n.ApplicationInformerSynced) {
		log.Error("timed out waiting for cache sync")
		return
	}
}
