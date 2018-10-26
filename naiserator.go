package naiserator

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/hashicorp/go-multierror"
	"github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	informers "github.com/nais/naiserator/pkg/client/informers/externalversions/naiserator/v1alpha1"
	r "github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/updater"
	corev1 "k8s.io/api/core/v1"
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
}

func NewNaiserator(clientSet kubernetes.Interface, appClient *clientV1Alpha1.Clientset, applicationInformer informers.ApplicationInformer) *Naiserator {
	naiserator := Naiserator{
		ClientSet:                 clientSet,
		AppClient:                 appClient,
		ApplicationInformer:       applicationInformer,
		ApplicationInformerSynced: applicationInformer.Informer().HasSynced}

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
	glog.Error(err)
	ev := app.CreateEvent(source, err.Error())
	_, err = n.reportEvent(ev)
	if err != nil {
		glog.Errorf("While creating an event for this error, another error occurred: %s", err)
	}
}

func (n *Naiserator) synchronize(previous, app *v1alpha1.Application) error {
	if err := v1alpha1.ApplyDefaults(app); err != nil {
		return fmt.Errorf("while applying default values to application spec: %s", err)
	}

	hash, err := app.Hash()
	if err != nil {
		return fmt.Errorf("while hashing application spec: %s", err)
	}
	if app.LastSyncedHash() == hash {
		glog.Infof("%s: no changes", app.Name)
		return nil
	}

	resources, err := r.Create(app)
	if err != nil {
		return fmt.Errorf("while creating resources: %s", err)
	}

	if err := n.createOrUpdateMany(resources); err != nil {
		return fmt.Errorf("while persisting resources to Kubernetes: %s", err)
	}

	app.SetLastSyncedHash(hash)
	glog.Infof("%s: setting new hash %s", app.Name, hash)

	app.NilFix()
	_, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(app)
	if err != nil {
		return fmt.Errorf("while storing application sync metadata: %s", err)
	}

	return nil
}

func (n *Naiserator) update(old, new interface{}) {
	var app, previous *v1alpha1.Application
	if old != nil {
		previous = old.(*v1alpha1.Application)
	}
	if new != nil {
		app = new.(*v1alpha1.Application)
	}

	glog.Infof("%s: synchronizing application", app.Name)

	if err := n.synchronize(previous, app); err != nil {
		glog.Errorf("%s: error %s", app.Name, err)
	} else {
		glog.Infof("%s: synchronized successfully", app.Name)
	}

	glog.Infof("%s: finished synchronizing", app.Name)
}

func (n *Naiserator) add(app interface{}) {
	n.update(nil, app)
}

func (n *Naiserator) createOrUpdateMany(resources []runtime.Object) error {
	var result = &multierror.Error{}

	for _, resource := range resources {
		err := updater.Updater(n.ClientSet, resource)()
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func (n *Naiserator) Run(stop <-chan struct{}) {
	if !cache.WaitForCacheSync(stop, n.ApplicationInformerSynced) {
		glog.Error("timed out waiting for cache sync")
		return
	}
}
