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
	"k8s.io/client-go/kubernetes"
)

func add(app *v1alpha1.Application, clientSet kubernetes.Interface, appClient clientV1Alpha1.NaisV1Alpha1Interface) {
	fmt.Println("added", app.Name)

	if !needsUpdate(app) {
		fmt.Println("nothing changed, skipping...")
		return
	}

	fmt.Println("updating...")
	//app.GenerateName = "test"
    //app.Annotations["last-succcessful-resourceversion"] = app.ResourceVersion

    result, err := appClient.Applications(app.Namespace).Update(app)
    if err != nil {
    	fmt.Println("error when updating annotation:", err)
		fmt.Printf("%+v\n", result)
	}
	//blockOwnerDeletion := true
	//svc := &k8score.Service{
	//	TypeMeta: metav1.TypeMeta{
	//		Kind:       "Service",
	//		APIVersion: "v1",
	//	},
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: app.Name,
	//		Namespace: "default",
	//		OwnerReferences: []metav1.OwnerReference{{
	//			Kind:               "Application",
	//			Name:               app.Name,
	//			APIVersion:         "v1alpha1",
	//			UID:                app.UID,
	//			BlockOwnerDeletion: &blockOwnerDeletion,
	//		}}},
	//	Spec: k8score.ServiceSpec{
	//		Ports: []k8score.ServicePort{{Port: 69}},
	//	},
	//}
	//
	//_, e := clientSet.CoreV1().Services("default").Create(svc)
	//
	//event := k8score.Event{ObjectMeta: metav1.ObjectMeta{GenerateName: "event"}, Action:"bananarama", ReportingInstance: "what?", Reason: "because", ReportingController:  "naiserator", InvolvedObject: app.GetObjectReference(), Message: "hallo", EventTime: metav1.MicroTime{Time: time.Now()}}
	//
	//_, err := clientSet.CoreV1().Events("default").Create(&event)
	//if err != nil {
	//	fmt.Println("error with event", err.Error())
	//}
	//
	//if e != nil {
	//	fmt.Println(e.Error())
	//}
}

func needsUpdate(app *v1alpha1.Application) bool {
	return app.Annotations["last-successful-resourceversion"] != app.ResourceVersion
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
