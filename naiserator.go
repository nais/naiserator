package main

import (
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
const APPLICATION_RESOURCE_VERSION = "nais.io/applicationResourceVersion"

// Returns true if a sub-resource's annotation matches the application's resource version.
func applicationResourceVersionSynced(app metav1.Object, subResource metav1.Object) bool {
	return subResource.GetAnnotations()[APPLICATION_RESOURCE_VERSION] == app.GetResourceVersion()
}

// Updates a sub-resource's application resource version annotation.
func updateResourceVersionAnnotations(app metav1.Object, subResource metav1.Object) {
	a := subResource.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[APPLICATION_RESOURCE_VERSION] = app.GetResourceVersion()
	subResource.SetAnnotations(a)
}

// Creates a Kubernetes event.
func reportEvent(event *corev1.Event, c kubernetes.Interface) (*corev1.Event, error) {
	return c.CoreV1().Events(event.Namespace).Create(event)
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
				Kind:               app.Kind,
				Name:               app.Name,
				APIVersion:         app.APIVersion,
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

func add(app *v1alpha1.Application, clientSet kubernetes.Interface, appClient clientV1Alpha1.NaisV1Alpha1Interface) {
	var err error

	glog.Infof("Start processing application '%s'", app.Name)

	glog.Infof("Querying service...")
	svc, err := clientSet.CoreV1().Services(app.Namespace).Get(app.Name, metav1.GetOptions{})
	if err == nil && applicationResourceVersionSynced(app, svc) {
		glog.Info("Service is already in sync with latest version of application spec, skipping.")
		return
	} else if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("Encountered an error while querying the Kubernetes API: %s", err)
		return
	}

	glog.Infof("Service needs update, starting synchronization...")
	_, err = createService(app, clientSet)
	if err != nil {
		glog.Errorf("While creating service: %s", err)
		ev := app.GenerateErrorEvent("createService", err.Error())
		_, err = reportEvent(ev, clientSet)
		if err != nil {
			glog.Errorf("Additionally, while creating an event for this error, another error occurred: %s", err)
		}
		return
	}
	glog.Info("Successfully created service.")

	glog.Info("Successfully processed application.")
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
				add(obj.(*v1alpha1.Application), genericClient, clientSet)
			},
			UpdateFunc: func(old, new interface{}) {
				add(new.(*v1alpha1.Application), genericClient, clientSet)
			},
		})

	go applicationInformer.Run(wait.NeverStop)
	return applicationStore
}
