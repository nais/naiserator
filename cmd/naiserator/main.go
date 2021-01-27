package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/Shopify/sarama"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/synchronizer"
	log "github.com/sirupsen/logrus"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func main() {
	err := run()

	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Info("Naiserator shutting down")
}

func Scheme(schemes ...func(*runtime.Scheme) error) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	for _, fn := range schemes {
		err := fn(scheme)
		if err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

func run() error {
	var err error

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)
	log.SetLevel(log.DebugLevel)

	log.Info("Naiserator starting up")

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

	kscheme, err := Scheme(
		clientgoscheme.AddToScheme,
		nais_io_v1alpha1.AddToScheme,
		nais_io_v1.AddToScheme,
		iam_cnrm_cloud_google_com_v1beta1.AddToScheme,
		sql_cnrm_cloud_google_com_v1beta1.AddToScheme,
		storage_cnrm_cloud_google_com_v1beta1.AddToScheme,
		networking_istio_io_v1alpha3.AddToScheme,
	)

	if err != nil {
		return err
	}

	metrics.Register(kubemetrics.Registry)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		SyncPeriod:         &cfg.Informer.FullSyncInterval,
		Scheme:             kscheme,
		MetricsBindAddress: cfg.Bind,
	})
	if err != nil {
		return err
	}

	stopCh := StopCh()

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

	syncerConfig := synchronizer.Config{
		KafkaEnabled:               cfg.Kafka.Enabled,
		QueueSize:                  cfg.Synchronizer.QueueSize,
		DeploymentMonitorFrequency: cfg.Synchronizer.RolloutCheckInterval,
		DeploymentMonitorTimeout:   cfg.Synchronizer.RolloutTimeout,
	}

	syncer := &synchronizer.Synchronizer{
		Client:          mgr.GetClient(),
		ResourceOptions: resourceOptions,
		Config:          syncerConfig,
	}

	if err = syncer.SetupWithManager(mgr); err != nil {
		return err
	}

	return mgr.Start(stopCh)
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
