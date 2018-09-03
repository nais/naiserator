package naiserator

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Kubernetes metadata annotation key used to store the version of the successfully processed resource.
const ApplicationResourceVersion = "nais.io/applicationResourceVersion"

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
func reportEvent(event *corev1.Event, c kubernetes.Interface) (*corev1.Event, error) {
	return c.CoreV1().Events(event.Namespace).Create(event)
}

// Reports an error through the error log, a Kubernetes event, and possibly logs a failure in event creation.
func reportError(source string, err error, app *v1alpha1.Application, c kubernetes.Interface) {
	glog.Error(err)
	ev := app.CreateEvent(source, err.Error())
	_, err = reportEvent(ev, c)
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

func createService(app *v1alpha1.Application, clientSet kubernetes.Interface) (*corev1.Service, error) {
	svc := generateService(app)
	updateResourceVersionAnnotations(app, svc)
	return clientSet.CoreV1().Services(app.Namespace).Create(svc)
}

func synchronizeService(clientSet kubernetes.Interface, app *v1alpha1.Application) error {
	svc, err := clientSet.CoreV1().Services(app.Namespace).Get(app.Name, metav1.GetOptions{})

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
		err = clientSet.CoreV1().Services(svc.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("while deleting service: %s", err)
		}
	}

	glog.Infof("Creating new service...")
	_, err = createService(app, clientSet)
	if err != nil {
		return fmt.Errorf("while creating service: %s", err)
	}

	return nil
}

func process(app *v1alpha1.Application, clientSet kubernetes.Interface) {
	var err error

	report := func(source string, err error) {
		reportError(source, err, app, clientSet)
	}

	glog.Infof("Start processing application '%s'", app.Name)

	glog.Infof("Start synchronizing service...")
	err = synchronizeService(clientSet, app)
	if err != nil {
		report("synchronizeService", err)
		return
	}
	glog.Infof("Successfully synchronized service.")

	glog.Info("Successfully processed application", app.Name)
}

func WatchResources(clientSet clientV1Alpha1.NaisV1Alpha1Interface, genericClient kubernetes.Interface) cache.Store {
	applicationStore, applicationInformer := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (result runtime.Object, err error) {
				return clientSet.Applications("default").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return clientSet.Applications("default").Watch(lo)
			},
		},
		&v1alpha1.Application{},
		1*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				process(obj.(*v1alpha1.Application), genericClient)
			},
			UpdateFunc: func(old, new interface{}) {
				process(new.(*v1alpha1.Application), genericClient)
			},
		})

	go applicationInformer.Run(wait.NeverStop)
	return applicationStore
}
