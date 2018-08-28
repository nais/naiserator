package main

import (
	"k8s.io/client-go/tools/cache"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"github.com/nais/naiserator/api/types/v1alpha1"
	"time"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
)

func add(app *v1alpha1.Application) {
	fmt.Println("added", app.Spec.Team)
}
func update(old, new *v1alpha1.Application) {
	fmt.Println("updated", old.Spec.Team, new.Spec.Team)
}
func delete(app *v1alpha1.Application) {
	fmt.Println("deleted", app.Spec.Team)
}

func WatchResources(clientSet clientV1Alpha1.NaisV1Alpha1Interface) cache.Store {
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
				add(obj.(*v1alpha1.Application))
			},
			UpdateFunc: func(old, new interface{}) {
				update(old.(*v1alpha1.Application), new.(*v1alpha1.Application))
			},
			DeleteFunc: func(obj interface{}) {
				delete(obj.(*v1alpha1.Application))
			}},
	)

	go applicationInformer.Run(wait.NeverStop)
	return applicationStore
}
