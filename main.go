package main

import (
	"flag"
	"log"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"fmt"

	"time"
)

var kubeconfig string

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.Parse()
}

func main() {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		log.Printf("using in-cluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		log.Printf("using configuration from '%s'", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		panic(err)
	}

	v1alpha1.AddToScheme(scheme.Scheme)

	clientSet, err := clientV1alpha1.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	naisClient := clientSet.Applications("default")
	applicationList, err := naisClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	application, e := naisClient.Get("nais-testapp", metav1.GetOptions{})

	if e != nil {
		panic(e)
	}

	fmt.Println("got nais deployment..", application.Spec.Team)

	fmt.Printf("applications found: %+v\n", applicationList)
	store := WatchResources(clientSet)

	for {
		applicationsFromStore := store.List()
		fmt.Printf("project in store: %d\n", len(applicationsFromStore))

		time.Sleep(2 * time.Second)
	}
}
