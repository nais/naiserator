package naiserator

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	"github.com/nais/naiserator/pkg/metrics"
	r "github.com/nais/naiserator/pkg/resourcecreator"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Naiserator struct {
	ClientSet kubernetes.Interface
	AppClient clientV1Alpha1.NaisV1Alpha1Interface
}

const LastSyncedHashAnnotation = "nais.io/lastSyncedHash"

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

func (n *Naiserator) update(old, new *v1alpha1.Application) {
	glog.Infoln("updating application", new.Name)

	hash, err := new.Hash()
	if err != nil {
		n.reportError("update, get hash", err, new)
	}

	// something has changed, synchronizing all resources
	if old.Annotations[LastSyncedHashAnnotation] != hash {
		n.synchronize(new)
		return
	}

	glog.Infoln("no changes detected in", new.Name, "skipping sync")
}

func (n *Naiserator) createOrUpdate(resources []runtime.Object) error {
	for _, resource := range resources {
		switch v := resource.(type) {
		case *corev1.Service:
			// check if resource exists (possibly generic?)
			// if it does, apply resourceversion and update. Else create
			fmt.Printf("updating service...")
			return nil
		default:
			fmt.Printf("I don't know about type %T!\n", v)
			return nil
		}
	}
	return nil
}

func (n *Naiserator) synchronize(app *v1alpha1.Application) {
	glog.Infoln("synchronizing application", app.Name)

	resources, err := r.CreateResourceSpecs(app)

	if err != nil {
		n.reportError("createResourceSpecs", err, app)
		return
	}

	if err := n.createOrUpdate(resources); err != nil {
		n.reportError("createOrUpdate(resources)", err, app)
		return
	}

	if err := n.setLastSynced(app); err != nil {
		n.reportError("setlastsyncedhash", err, app)
	}

	metrics.ApplicationsSynchronized.Inc()
	glog.Infoln("successfully synchronized application", app.Name)
}

func (n *Naiserator) setLastSynced(app *v1alpha1.Application) error {
	hash, err := app.Hash()
	if err != nil {
		return err
	}

	glog.Infoln("setting last synced hash annotation to", hash)
	app.Annotations[LastSyncedHashAnnotation] = hash
	_, err = n.AppClient.Applications(app.Namespace).Update(app)
	return err
}

func (n *Naiserator) WatchResources() cache.Store {
	applicationStore, applicationInformer := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (result runtime.Object, err error) {
				return n.AppClient.Applications("default").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return n.AppClient.Applications("default").Watch(lo)
			},
		},
		&v1alpha1.Application{},
		1*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				n.synchronize(obj.(*v1alpha1.Application))
			},
			UpdateFunc: func(old, new interface{}) {
				n.update(old.(*v1alpha1.Application), new.(*v1alpha1.Application))
			},
		})

	go applicationInformer.Run(wait.NeverStop)
	return applicationStore
}
