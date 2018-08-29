package main

import (
	"time"

	"github.com/golang/glog"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Kubernetes metadata annotation key used to store the version of the successfully processed resource.
const LAST_SUCCESSFUL_RESOURCE_VERSION_KEY = "nais.io/lastSuccessfulResourceVersion"

func needsUpdate(app *v1alpha1.Application) bool {
	return app.Annotations[LAST_SUCCESSFUL_RESOURCE_VERSION_KEY] != app.ResourceVersion
}

func reportEvent(event *corev1.Event, c kubernetes.Interface) (*corev1.Event, error) {
	return c.CoreV1().Events(event.Namespace).Create(event)
}

func createService(app *v1alpha1.Application, clientSet kubernetes.Interface) (*corev1.Service, error) {
	blockOwnerDeletion := true
	svc := &corev1.Service{
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

	return clientSet.CoreV1().Services(app.Namespace).Create(svc)
}

func updateAnnotation(app *v1alpha1.Application, appClient clientV1Alpha1.NaisV1Alpha1Interface) (*v1alpha1.Application, error) {
	if app.Annotations == nil {
		app.Annotations = make(map[string]string)
	}
	app.Annotations[LAST_SUCCESSFUL_RESOURCE_VERSION_KEY] = app.ResourceVersion
	return appClient.Applications(app.Namespace).Update(app)
}

func add(app *v1alpha1.Application, clientSet kubernetes.Interface, appClient clientV1Alpha1.NaisV1Alpha1Interface) {
	var err error

	glog.Infof("Start processing application '%s'", app.Name)

	if !needsUpdate(app) {
		glog.Info("Application has not changed since last successful iteration, skipping.")
		return
	}

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

	_, err = updateAnnotation(app, appClient)
	if err != nil {
		glog.Errorf("While updating annotation: %s", err)
		return
	}

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
