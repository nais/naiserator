package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	istioClient "istio.io/client-go/pkg/clientset/versioned"

	"github.com/Shopify/sarama"
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	clientset "github.com/nais/naiserator/pkg/client/clientset/versioned"
	informers "github.com/nais/naiserator/pkg/client/informers/externalversions"
	"github.com/nais/naiserator/pkg/informer"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/synchronizer"
	log "github.com/sirupsen/logrus"
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

	log.Info("Synchronizer shutting down")
}

func run() error {
	var err error

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)
	log.SetLevel(log.DebugLevel)

	log.Info("Synchronizer starting up")

	cfg, err := config.New()
	if err != nil {
		return err
	}

	config.Print([]string{
		"kafka.sasl.username",
		"kafka.sasl.password",
	})

	if cfg.Kafka.Enabled {
		kafkaLogger := log.New()
		kafkaLogger.Level, err = log.ParseLevel(cfg.Kafka.LogVerbosity)
		if err != nil {
			log.Fatalf("while setting log level: %s", err)
		}
		kafkaLogger.SetLevel(log.GetLevel())
		kafkaLogger.SetFormatter(&formatter)
		sarama.Logger = kafkaLogger

		kafkaClient, err := kafka.NewClient(&cfg.Kafka)
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

	kubeconfig, err := getK8sConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize kubernetes config: %s", err)
	}

	// serve metrics
	go metrics.Serve(
		cfg.Bind,
		"/metrics",
		"/ready",
		"/alive",
	)

	resourceOptions := resourcecreator.NewResourceOptions()
	resourceOptions.AccessPolicy = cfg.Features.AccessPolicy
	resourceOptions.AccessPolicyNotAllowedCIDRs = cfg.Features.AccessPolicyNotAllowedCIDRs
	resourceOptions.NativeSecrets = cfg.Features.NativeSecrets
	resourceOptions.GoogleProjectId = cfg.GoogleProjectId
	resourceOptions.ClusterName = cfg.ClusterName
	resourceOptions.JwkerEnabled = cfg.Features.Jwker
	resourceOptions.JwkerServiceEntryHosts = cfg.ServiceEntryHosts.Jwker
	resourceOptions.AzureratorEnabled = cfg.Features.Azurerator
	resourceOptions.AzureratorServiceEntryHosts = cfg.ServiceEntryHosts.Azurerator
	resourceOptions.KafkaratorEnabled = cfg.Features.Kafkarator
	resourceOptions.DigdiratorEnabled = cfg.Features.Digdirator
	resourceOptions.DigdiratorServiceEntryHosts = cfg.ServiceEntryHosts.Digdirator
	resourceOptions.HostAliases = cfg.HostAliases
	resourceOptions.GatewayMappings = cfg.GatewayMappings
	resourceOptions.ApiServerIp = cfg.ApiServerIp

	if len(resourceOptions.GoogleProjectId) > 0 && len(resourceOptions.GatewayMappings) == 0 {
		return fmt.Errorf("running in GCP and no gateway mappings defined. Will not be able to set the right gateway on the Virtual Service based on the provided ingresses")
	}

	applicationInformerFactory := createApplicationInformerFactory(kubeconfig, cfg.Informer.FullSyncInterval)
	applicationClientset := createApplicationClientset(kubeconfig)
	istioClient := createIstioClientset(kubeconfig)
	genericClientset := createGenericClientset(kubeconfig)

	syncerConfig := synchronizer.Config{
		KafkaEnabled:               cfg.Kafka.Enabled,
		QueueSize:                  cfg.Synchronizer.QueueSize,
		DeploymentMonitorFrequency: cfg.Synchronizer.RolloutCheckInterval,
		DeploymentMonitorTimeout:   cfg.Synchronizer.RolloutTimeout,
	}

	syncer := synchronizer.New(
		genericClientset,
		applicationClientset,
		istioClient,
		resourceOptions,
		syncerConfig,
	)

	inf := informer.New(syncer, applicationInformerFactory)

	err = inf.Run()
	if err != nil {
		return fmt.Errorf("unable to start informer: %s", err)
	}

	go syncer.Main()
	<-stopCh

	inf.Stop()

	return nil
}

func createApplicationInformerFactory(kubeconfig *rest.Config, interval time.Duration) informers.SharedInformerFactory {
	config, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create naiserator clientset: %s", err)
	}

	return informers.NewSharedInformerFactory(config, interval)
}

func createApplicationClientset(kubeconfig *rest.Config) *clientset.Clientset {
	clientSet, err := clientset.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create application clientset: %s", err)
	}

	return clientSet
}

func createIstioClientset(kubeconfig *rest.Config) *istioClient.Clientset {
	clientSet, err := istioClient.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create istio clientset: %s", err)
	}

	return clientSet
}

func createGenericClientset(kubeconfig *rest.Config) *kubernetes.Clientset {
	cs, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Fatalf("unable to create generic clientset: %s", err)
	}

	return cs
}

func getK8sConfig(cfg *config.Config) (conf *rest.Config, err error) {
	kubeconfig := cfg.Kubeconfig
	if kubeconfig == "" {
		log.Infof("using in-cluster configuration")
		conf, err = rest.InClusterConfig()
	} else {
		log.Infof("using configuration from '%s'", kubeconfig)
		conf, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if conf != nil {
		conf.Burst = cfg.Ratelimit.Burst
		conf.QPS = float32(cfg.Ratelimit.QPS)
	}
	return
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
