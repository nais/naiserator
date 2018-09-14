package naiserator

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	"github.com/golang/glog"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	"github.com/nais/naiserator/pkg/metrics"
	r "github.com/nais/naiserator/pkg/resourcecreator"
	appsv1 "k8s.io/api/apps/v1"
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

func (n *Naiserator) synchronize(app *v1alpha1.Application) {
	glog.Infoln("synchronizing application", app.Name)

	if err := v1alpha1.ApplyDefaults(app); err != nil {
		glog.Errorf("Could not merge application struct with defaults %s", err)
		return
	}

	resources, err := r.GetResources(app)

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

func (n *Naiserator) createOrUpdate(resources []runtime.Object) error {
	var result = &multierror.Error{}

	for _, resource := range resources {
		switch r := resource.(type) {
		case *corev1.Service:
			svcClient := n.ClientSet.CoreV1().Services(r.Namespace)
			svc, err := svcClient.Get(r.Name, metav1.GetOptions{})

			// we have an existing resource, append resourceversion and update
			if err == nil {
				r.ObjectMeta.ResourceVersion = svc.ObjectMeta.ResourceVersion
				r.Spec.ClusterIP = svc.Spec.ClusterIP // ClusterIP must be retained as the field is immutable
				if _, err := svcClient.Update(r); err != nil {
					multierror.Append(result, fmt.Errorf("unable to update service: %s", err))
				}
				continue
			}

			// no resources found, creating a new one
			if errors.IsNotFound(err) {
				if _, err := svcClient.Create(r); err != nil {
					multierror.Append(result, fmt.Errorf("unable to create service: %s", err))
				}
				continue
			}

			multierror.Append(result, fmt.Errorf("unable to synchronize service: %s", err))
			continue
		case *appsv1.Deployment:
			deployClient := n.ClientSet.AppsV1().Deployments(r.Namespace)
			deploy, err := deployClient.Get(r.Name, metav1.GetOptions{})

			// we have an existing resource, append resourceversion and update
			if err == nil {
				r.ObjectMeta.ResourceVersion = deploy.ObjectMeta.ResourceVersion
				if _, err := deployClient.Update(r); err != nil {
					multierror.Append(result, fmt.Errorf("unable to update deployment: %s", err))
				}
				continue
			}

			// no resources found, creating a new one
			if errors.IsNotFound(err) {
				if _, err := deployClient.Create(r); err != nil {
					multierror.Append(result, fmt.Errorf("unable to create deployment: %s", err))
				}
				continue
			}

			multierror.Append(result, fmt.Errorf("unable to synchronize deployment: %s", err))
			continue
		default:
			fmt.Printf("unknown type %T\n", r)
			return nil
		}
	}

	return result.ErrorOrNil()
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
				return n.AppClient.Applications("").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return n.AppClient.Applications("").Watch(lo)
			},
		},
		&v1alpha1.Application{},
		5*time.Minute,
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
