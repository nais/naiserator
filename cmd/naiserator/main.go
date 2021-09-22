package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/nais/liberator/pkg/tlsutil"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/nais/naiserator/pkg/controllers"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"

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

	if cfg.Features.Vault {
		err = cfg.Vault.Validate()
		if err != nil {
			return err
		}
	}

	var kafkaClient kafka.Interface

	if cfg.Kafka.Enabled {
		kafkaLogger := log.New()
		kafkaLogger.Level, err = log.ParseLevel(cfg.Kafka.LogVerbosity)
		if err != nil {
			log.Fatalf("while setting log level: %s", err)
		}
		kafkaLogger.SetLevel(log.GetLevel())
		kafkaLogger.SetFormatter(&formatter)

		kafkaTLS := &tls.Config{}
		if cfg.Kafka.TLS.Enabled {
			kafkaTLS, err = tlsutil.TLSConfigFromFiles(cfg.Kafka.TLS.CertificatePath, cfg.Kafka.TLS.PrivateKeyPath, cfg.Kafka.TLS.CAPath)
			if err != nil {
				log.Fatalf("load Kafka TLS credentials: %s", err)
			}
		}

		kafkaClient, err = kafka.New(cfg.Kafka.Brokers, cfg.Kafka.Topic, kafkaTLS, kafkaLogger)
		if err != nil {
			log.Fatalf("unable to setup kafka: %s", err)
		}
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

	if cfg.Features.Webhook {
		// Register create/update validation webhooks for liberator_scheme's CRDs
		if err := liberator_scheme.Webhooks(mgr); err != nil {
			return err
		}
	}

	resourceOptions := resource.NewOptions()
	resourceOptions.AccessPolicyNotAllowedCIDRs = cfg.Features.AccessPolicyNotAllowedCIDRs
	resourceOptions.ApiServerIp = cfg.ApiServerIp
	resourceOptions.AzureratorEnabled = cfg.Features.Azurerator
	resourceOptions.ClusterName = cfg.ClusterName
	resourceOptions.DigdiratorEnabled = cfg.Features.Digdirator
	resourceOptions.DigdiratorHosts = cfg.ServiceHosts.Digdirator
	resourceOptions.GatewayMappings = cfg.GatewayMappings
	resourceOptions.GoogleCloudSQLProxyContainerImage = cfg.GoogleCloudSQLProxyContainerImage
	resourceOptions.GoogleProjectId = cfg.GoogleProjectId
	resourceOptions.HostAliases = cfg.HostAliases
	resourceOptions.JwkerEnabled = cfg.Features.Jwker
	resourceOptions.CNRMEnabled = cfg.Features.CNRM
	resourceOptions.KafkaratorEnabled = cfg.Features.Kafkarator
	resourceOptions.NativeSecrets = cfg.Features.NativeSecrets
	resourceOptions.NetworkPolicy = cfg.Features.NetworkPolicy
	resourceOptions.Proxy = cfg.Proxy
	resourceOptions.Securelogs = cfg.Securelogs
	resourceOptions.SecurePodSecurityContext = cfg.Features.SecurePodSecurityContext
	resourceOptions.VaultEnabled = cfg.Features.Vault
	resourceOptions.Vault = cfg.Vault
	resourceOptions.Wonderwall = cfg.Wonderwall


	if cfg.Features.GCP && len(resourceOptions.GatewayMappings) == 0 {
		return fmt.Errorf("running in GCP and no gateway mappings defined. Will not be able to set the right gateway on the ingress")
	}

	mgrClient := mgr.GetClient()
	simpleClient, err := client.New(kconfig, client.Options{
		Scheme: kscheme,
	})
	if err != nil {
		return err
	}

	if cfg.DryRun {
		mgrClient = readonly.NewClient(mgrClient)
		simpleClient = readonly.NewClient(simpleClient)
	}

	applicationReconciler := controllers.NewAppReconciler(synchronizer.Synchronizer{
		Client:          mgrClient,
		Config:          *cfg,
		Kafka:           kafkaClient,
		ResourceOptions: resourceOptions,
		RolloutMonitor:  make(map[client.ObjectKey]synchronizer.RolloutMonitor),
		Scheme:          kscheme,
		SimpleClient:    simpleClient,
	})

	if err = applicationReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	naisjobReconciler := controllers.NewNaisjobReconciler(synchronizer.Synchronizer{
		Client:          mgrClient,
		Config:          *cfg,
		Kafka:           kafkaClient,
		ResourceOptions: resourceOptions,
		RolloutMonitor:  make(map[client.ObjectKey]synchronizer.RolloutMonitor),
		Scheme:          kscheme,
		SimpleClient:    simpleClient,
	})

	if err = naisjobReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}
