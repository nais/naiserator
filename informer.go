package main

import (
	"time"

	"github.com/jhrv/operator/tutorial/api/types/v1alpha1"
	client_v1alpha1 "github.com/jhrv/operator/tutorial/clientset/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func WatchResources(clientSet client_v1alpha1.NaisV1Alpha1Interface) cache.Store {
	naisDeploymentStore, naisDeploymentController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(lo metav1.ListOptions) (result runtime.Object, err error) {
				return clientSet.NaisDeployments("default").List(lo)
			},
			WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
				return clientSet.NaisDeployments("default").Watch(lo)
			},
		},
		&v1alpha1.NaisDeployment{},
		1*time.Minute,
		cache.ResourceEventHandlerFuncs{},
	)

	go naisDeploymentController.Run(wait.NeverStop)
	return naisDeploymentStore
}
