package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/nais/naiserator/pkg/resourcecreator/resourceutils"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/readonly"
	"github.com/nais/naiserator/pkg/synchronizer"
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

	// Register CRDs with controller-tools
	kscheme, err := liberator_scheme.All()
	if err != nil {
		return err
	}

	kconfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	kconfig.QPS = float32(cfg.Ratelimit.QPS)
	kconfig.Burst = cfg.Ratelimit.Burst

	metrics.Register(kubemetrics.Registry)
	mgr, err := ctrl.NewManager(kconfig, ctrl.Options{
		SyncPeriod:         &cfg.Informer.FullSyncInterval,
		Scheme:             kscheme,
		MetricsBindAddress: cfg.Bind,
	})
	if err != nil {
		return err
	}

	stopCh := StopCh()

	resourceOptions := resourceutils.NewResourceOptions()
	resourceOptions.AccessPolicyNotAllowedCIDRs = cfg.Features.AccessPolicyNotAllowedCIDRs
	resourceOptions.ApiServerIp = cfg.ApiServerIp
	resourceOptions.AzureratorEnabled = cfg.Features.Azurerator
	resourceOptions.AzureratorHosts = cfg.ServiceHosts.Azurerator
	resourceOptions.ClusterName = cfg.ClusterName
	resourceOptions.DigdiratorEnabled = cfg.Features.Digdirator
	resourceOptions.DigdiratorHosts = cfg.ServiceHosts.Digdirator
	resourceOptions.GatewayMappings = cfg.GatewayMappings
	resourceOptions.GoogleProjectId = cfg.GoogleProjectId
	resourceOptions.HostAliases = cfg.HostAliases
	resourceOptions.JwkerEnabled = cfg.Features.Jwker
	resourceOptions.JwkerHosts = cfg.ServiceHosts.Jwker
	resourceOptions.KafkaratorEnabled = cfg.Features.Kafkarator
	resourceOptions.NativeSecrets = cfg.Features.NativeSecrets
	resourceOptions.NetworkPolicy = cfg.Features.NetworkPolicy

	if len(resourceOptions.GoogleProjectId) > 0 && len(resourceOptions.GatewayMappings) == 0 {
		return fmt.Errorf("running in GCP and no gateway mappings defined. Will not be able to set the right gateway on the ingress.")
	}

	mgrClient := mgr.GetClient()
	simpleClient, err := client.New(kconfig, client.Options{
		Scheme: kscheme,
	})

	if cfg.DryRun {
		mgrClient = readonly.NewClient(mgrClient)
		simpleClient = readonly.NewClient(simpleClient)
	}

	syncer := &synchronizer.Synchronizer{
		Client:                 mgrClient,
		SimpleClient:           simpleClient,
		Scheme:                 kscheme,
		ResourceOptions:        resourceOptions,
		Config:                 *cfg,
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
