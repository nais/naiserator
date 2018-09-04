package naiserator

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	"github.com/nais/naiserator/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// Kubernetes metadata annotation key used to store the version of the successfully processed resource.
const ApplicationResourceVersion = "nais.io/applicationResourceVersion"
const LastSyncedHashAnnotation = "nais.io/lastSyncedHash"

// Returns true if a sub-resource's annotation matches the application's resource version.
func applicationResourceVersionSynced(app metav1.Object, subResource metav1.Object) bool {
	return subResource.GetAnnotations()[ApplicationResourceVersion] == app.GetResourceVersion()
}

// Updates a sub-resource's application resource version annotation.
func updateResourceVersionAnnotations(app metav1.Object, subResource metav1.Object) {
	a := subResource.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[ApplicationResourceVersion] = app.GetResourceVersion()
	subResource.SetAnnotations(a)
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

func generateService(app *v1alpha1.Application) *corev1.Service {
	blockOwnerDeletion := true
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "v1alpha1",
				Kind:               "Application",
				Name:               app.Name,
				UID:                app.UID,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}}},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 69,
				},
			},
		},
	}
}

func (n *Naiserator) createService(app *v1alpha1.Application) (*corev1.Service, error) {
	svc := generateService(app)
	updateResourceVersionAnnotations(app, svc)
	return n.ClientSet.CoreV1().Services(app.Namespace).Create(svc)
}

func (n *Naiserator) synchronizeService(app *v1alpha1.Application) error {
	svc, err := n.ClientSet.CoreV1().Services(app.Namespace).Get(app.Name, metav1.GetOptions{})

	if err == nil && applicationResourceVersionSynced(app, svc) {
		glog.Info("Service is already in sync with latest version of application spec, skipping.")
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("while querying the Kubernetes API: %s", err)
	}

	// should we delete, or simply update like before?
	if svc != nil && !errors.IsNotFound(err) {
		glog.Infof("Deleting old service...")
		err = n.ClientSet.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("while deleting service: %s", err)
		}
	}

	glog.Infof("Creating new service...")
	_, err = n.createService(app)
	if err != nil {
		return fmt.Errorf("while creating service: %s", err)
	}
	metrics.ResourcesGenerated.Inc()

	return nil
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

	var err error

	report := func(source string, err error) {
		n.reportError(source, err, app)
	}

	glog.Infof("Start processing application '%s'", app.Name)
	metrics.ResourcesProcessed.Inc()

	glog.Infof("Start synchronizing service...")
	err = n.synchronizeService(app)
	if err != nil {
		report("synchronizeService", err)
		return
	}
	glog.Infof("Successfully synchronized service.")

	glog.Info("Successfully processed application", app.Name)

	if err := n.setLastSynced(app); err != nil {
		n.reportError("setlastsyncedhash", err, app)
	}
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
