package naiserator

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/hashicorp/go-multierror"
	"github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	"github.com/nais/naiserator/pkg/metrics"
	r "github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/updater"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Naiserator is a singleton that holds Kubernetes client instances.
type Naiserator struct {
	ClientSet kubernetes.Interface
	AppClient *clientV1Alpha1.Clientset
}

const lastSyncedHashAnnotation = "nais.io/lastSyncedHash"

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

func (n *Naiserator) update(old, new *v1alpha1.Application) error {
	glog.Infof("%s: updating application", new.Name)

	hash, err := new.Hash()
	if err != nil {
		err = fmt.Errorf("while hashing application spec: %s", err)
		n.reportError("%s %+v", err, new)
		return err
	}

	// something has changed, synchronizing all resources
	if old.Annotations[lastSyncedHashAnnotation] != hash {
		glog.Infof("%s: changes detected", new.Name)
		return n.synchronize(new)
	}

	glog.Infof("%s: no changes detected, skipping sync", new.Name)

	return nil
}

func (n *Naiserator) synchronize(app *v1alpha1.Application) error {
	glog.Infof("%s: synchronizing application", app.Name)

	if err := v1alpha1.ApplyDefaults(app); err != nil {
		return fmt.Errorf("%s: could not merge application struct with defaults", err)
	}

	resources, err := r.Create(app)

	if err != nil {
		n.reportError("createResourceSpecs", err, app)
		return err
	}

	if err := n.createOrUpdateMany(resources); err != nil {
		n.reportError("createOrUpdateMany(resources)", err, app)
		return err
	}

	if err := n.setLastSynced(app); err != nil {
		n.reportError("setlastsyncedhash", err, app)
	}

	metrics.ApplicationsSynchronized.Inc()
	glog.Infof("%s: successfully synchronized application", app.Name)

	return nil
}

func (n *Naiserator) createOrUpdateMany(resources []runtime.Object) error {
	var result = &multierror.Error{}

	for _, resource := range resources {
		err := updater.Updater(n.ClientSet, resource)()
		multierror.Append(err)
	}

	return result.ErrorOrNil()
}

func (n *Naiserator) setLastSynced(app *v1alpha1.Application) error {
	hash, err := app.Hash()
	if err != nil {
		return err
	}

	glog.Infof("%s: setting last synced hash annotation to %x", app.Name, hash)
	if app.Annotations == nil {
		app.Annotations = make(map[string]string)
	}
	app.Annotations[lastSyncedHashAnnotation] = hash
	_, err = n.AppClient.NaiseratorV1alpha1().Applications(app.Namespace).Update(app)
	return err
}

// WatchResources is the Naiserator main loop, which
// synchronizes Application specs to Kubernetes resources indefinitely.
func (n *Naiserator) WatchResources() cache.Store {
	applicationStore, applicationInformer := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (result runtime.Object, err error) {
				return n.AppClient.NaiseratorV1alpha1().Applications("").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return n.AppClient.NaiseratorV1alpha1().Applications("").Watch(lo)
			},
		},
		&v1alpha1.Application{},
		5*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				err := n.synchronize(obj.(*v1alpha1.Application))
				if err != nil {
					glog.Errorln(err)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				err := n.update(old.(*v1alpha1.Application), new.(*v1alpha1.Application))
				if err != nil {
					glog.Errorln(err)
				}
			},
		})

	go applicationInformer.Run(wait.NeverStop)
	return applicationStore
}
