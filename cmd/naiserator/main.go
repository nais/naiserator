package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	fqdn_scheme "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/liberator/pkg/tlsutil"
	pov1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/nais/naiserator/pkg/controllers"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/kafka"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/readonly"
	naiserator_scheme "github.com/nais/naiserator/pkg/scheme"
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

	if cfg.FQDNPolicy.Enabled {
		err := fqdn_scheme.AddToScheme(kscheme)
		if err != nil {
			return err
		}
	}

	if cfg.Features.PrometheusOperator {
		err = pov1.AddToScheme(kscheme)
		if err != nil {
			return err
		}
	}

	kconfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	kconfig.QPS = float32(cfg.Ratelimit.QPS)
	kconfig.Burst = cfg.Ratelimit.Burst

	metrics.Register(kubemetrics.Registry)
	mgr, err := ctrl.NewManager(kconfig, ctrl.Options{
		Cache: cache.Options{
			SyncPeriod: &cfg.Informer.FullSyncInterval,
		},
		Scheme: kscheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.Bind,
		},
		HealthProbeBindAddress: cfg.HealthProbeBindAddress,
	})
	if err != nil {
		return err
	}

	// make us immediately healthy
	err = mgr.AddReadyzCheck("ready", func(req *http.Request) error { return nil })
	if err != nil {
		return err
	}

	if cfg.Features.Webhook {
		// Register create/update validation webhooks for liberator_scheme's CRDs
		if err := liberator_scheme.Webhooks(mgr); err != nil {
			return err
		}
	}

	if len(cfg.GatewayMappings) == 0 {
		return fmt.Errorf("no gateway mappings defined. Will not be able to set the right gateway on the ingress")
	}

	listers := naiserator_scheme.GenericListers()
	if len(cfg.GoogleProjectId) > 0 {
		listers = append(listers, naiserator_scheme.GCPListers()...)
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

	applicationReconciler := controllers.NewAppReconciler(synchronizer.NewSynchronizer(
		mgrClient,
		simpleClient,
		*cfg,
		&generators.Application{
			Config: *cfg,
		},
		kafkaClient,
		listers,
		kscheme,
	))

	opts := []controllers.Option{
		controllers.WithMaxConcurrentReconciles(cfg.MaxConcurrentReconciles),
	}

	err = applicationReconciler.SetupWithManager(mgr, opts...)
	if err != nil {
		return err
	}

	naisjobReconciler := controllers.NewNaisjobReconciler(synchronizer.NewSynchronizer(
		mgrClient,
		simpleClient,
		*cfg,
		&generators.Naisjob{
			Config: *cfg,
		},
		kafkaClient,
		listers,
		kscheme,
	))

	err = naisjobReconciler.SetupWithManager(mgr, opts...)
	if err != nil {
		return err
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}
