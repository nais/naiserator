package main

import (
	"flag"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"syscall"
	"github.com/golang/glog"
	"github.com/nais/naiserator/api/types/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
)

var kubeconfig string

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.Parse()
}

func main() {
	glog.Info("starting up")

    // register custom types
	v1alpha1.AddToScheme(scheme.Scheme)

	// make stop channel for exit signals
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	WatchResources(createClientSet())
	<-s

	glog.Info("shutting down")
}

func createClientSet() (*clientV1Alpha1.NaisV1Alpha1Client) {
	config, err := getK8sConfig()
	if err != nil {
	   glog.Fatalf("unable to initialize kubernetes config")
	}
	clientSet, err := clientV1Alpha1.NewForConfig(config)

	if err != nil {
		glog.Fatalf("unable to create new clientset")
	}
	return clientSet
}

func getK8sConfig() (*rest.Config, error) {
	if kubeconfig == "" {
		glog.Infof("using in-cluster configuration")
		return rest.InClusterConfig()
	} else {
		glog.Infof("using configuration from '%s'", kubeconfig)
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
}

