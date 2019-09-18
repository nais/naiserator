package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	clientV1Alpha1 "github.com/nais/naiserator/pkg/client/clientset/versioned"
	clientset "github.com/nais/naiserator/pkg/client/clientset/versioned"
	informers "github.com/nais/naiserator/pkg/client/informers/externalversions"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/synchronizer"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	err := run()

	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Info("Naiserator shutting down")
}

func run() error {
	var err error

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)

	log.Info("Naiserator starting up")

	config, err := configuration()
	if err != nil {
		return err
	}

	printConfig([]string{})

	if config.Kafka.Enabled {
		kafkaLogger := log.New()
		kafkaLogger.Level, err = log.ParseLevel(config.Kafka.LogVerbosity)
		if err != nil {
			log.Fatalf("while setting log level: %s", err)
		}
		kafkaLogger.SetLevel(log.GetLevel())
		kafkaLogger.SetFormatter(&formatter)
		sarama.Logger = kafkaLogger

		kafkaClient, err := kafka.NewClient(&config.Kafka)
		if err != nil {
			log.Fatalf("unable to setup kafka: %s", err)
		}
		go kafkaClient.ProducerLoop()
	}

	// register custom types
	err = v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return fmt.Errorf("unable to add scheme: %s", err)
	}

	stopCh := StopCh()

	kubeconfig, err := getK8sConfig()
	if err != nil {
		return fmt.Errorf("unable to initialize kubernetes config: %s", err)
	}

	// serve metrics
	go metrics.Serve(
		viper.GetString("bind"),
		"/metrics",
		"/ready",
		"/alive",
	)

	resourceOptions := resourcecreator.NewResourceOptions()
	resourceOptions.AccessPolicy = viper.GetBool("features.access-policy")
	resourceOptions.NativeSecrets = viper.GetBool("features.native-secrets")

	applicationInformerFactory := createApplicationInformerFactory(kubeconfig)
	n := synchronizer.New(
		createGenericClientset(kubeconfig),
		createApplicationClientset(kubeconfig),
		applicationInformerFactory.Naiserator().V1alpha1().Applications(),
		resourceOptions,
		config.Kafka.Enabled)

	applicationInformerFactory.Start(stopCh)
	n.Run(stopCh)
	<-stopCh

	return nil
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
	kubeconfig := viper.GetString("kubeconfig")
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
