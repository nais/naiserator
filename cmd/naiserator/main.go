package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/nais/naiserator"
	"github.com/nais/naiserator/api/types/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/clientset/v1alpha1"
	"github.com/nais/naiserator/metrics"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
	bindAddr   string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.Parse()
}

func main() {
	glog.Info("starting up")

	// register custom types
	v1alpha1.AddToScheme(scheme.Scheme)

	// make stop channel for exit signals
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	kubeconfig, err := getK8sConfig()
	if err != nil {
		glog.Fatalf("unable to initialize kubernetes config")
	}

	// serve metrics
	go metrics.Serve(
		bindAddr,
		"/metrics",
		"/ready",
		"/alive",
	)

	naiserator.Naiserator{ClientSet: createGenericClient(kubeconfig), AppClient: createApplicationClient(kubeconfig)}.WatchResources()

	<-s

	glog.Info("shutting down")
}

func createApplicationClient(kubeconfig *rest.Config) *clientV1Alpha1.NaisV1Alpha1Client {
	clientSet, err := clientV1Alpha1.NewForConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("unable to create new clientset")
	}

	return clientSet
}

func createGenericClient(kubeconfig *rest.Config) *kubernetes.Clientset {
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return clientset
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
