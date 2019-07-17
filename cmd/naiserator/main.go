package main

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nais/naiserator"
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	clientset "github.com/nais/naiserator/pkg/client/clientset/versioned"
	informers "github.com/nais/naiserator/pkg/client/informers/externalversions"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig           string
	bindAddr             string
	accessPolicyEnabled  bool
	nativeSecretsEnabled bool

	kafkaConfig = kafka.DefaultConfig()
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "path to Kubernetes config file")
	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.BoolVar(&accessPolicyEnabled, "access-policy-enabled", ensureBool(getEnv("ACCESS_POLICY_ENABLED", "false")), "enable access policy with Istio and NetworkPolicies")
	flag.BoolVar(&nativeSecretsEnabled, "native-secrets-enabled", ensureBool(getEnv("NATIVE_SECRETS_ENABLED", "false")), "enable use of native secrets")

	kafka.SetupFlags(&kafkaConfig)
	flag.Parse()
}

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	log.Info("Naiserator starting up")

	if kafkaConfig.Enabled {
		kafkaClient, err := kafka.NewClient(&kafkaConfig)
		if err != nil {
			log.Fatalf("unable to setup kafka: %s", err)
		}
		go kafkaClient.ProducerLoop()
	}

	// register custom types
	err := v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		log.Fatal("unable to add scheme")
	}

	stopCh := StopCh()

	kubeconfig, err := getK8sConfig()
	if err != nil {
		log.Fatal("unable to initialize kubernetes config")
	}

	// serve metrics
	go metrics.Serve(
		bindAddr,
		"/metrics",
		"/ready",
		"/alive",
	)

	resourceOptions := resourcecreator.NewResourceOptions()
	resourceOptions.AccessPolicy = accessPolicyEnabled
	resourceOptions.NativeSecrets = nativeSecretsEnabled

	applicationInformerFactory := createApplicationInformerFactory(kubeconfig)
	n := naiserator.NewNaiserator(
		createGenericClientset(kubeconfig),
		createApplicationClientset(kubeconfig),
		applicationInformerFactory.Naiserator().V1alpha1().Applications(),
		resourceOptions,
		kafkaConfig.Enabled)

	applicationInformerFactory.Start(stopCh)
	n.Run(stopCh)
	<-stopCh

	log.Info("Naiserator has shut down")
}

func createApplicationInformerFactory(kubeconfig *rest.Config) informers.SharedInformerFactory {
	config, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create naiserator clientset: %s", err)
	}

	return informers.NewSharedInformerFactory(config, time.Second*30)
}

func createApplicationClientset(kubeconfig *rest.Config) *clientV1Alpha1.Clientset {
	clientSet, err := clientV1Alpha1.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create new clientset")
	}

	return clientSet
}

func createGenericClientset(kubeconfig *rest.Config) *kubernetes.Clientset {
	cs, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return cs
}

func getK8sConfig() (*rest.Config, error) {
	if kubeconfig == "" {
		log.Infof("using in-cluster configuration")
		return rest.InClusterConfig()
	} else {
		log.Infof("using configuration from '%s'", kubeconfig)
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
}

func StopCh() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func ensureBool(str string) bool {
	bool, err := strconv.ParseBool(str)

	if err != nil {
		log.Errorf("unable to parse boolean \"%s\", defaulting to false", str)
	}

	return bool
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
