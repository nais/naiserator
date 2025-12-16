package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	fqdn_scheme "github.com/nais/liberator/pkg/apis/fqdnnetworkpolicies.networking.gke.io/v1alpha3"
	"github.com/nais/liberator/pkg/logrus2logr"
	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/naiserator/pkg/controllers"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/readonly"
	naiserator_scheme "github.com/nais/naiserator/pkg/scheme"
	"github.com/nais/naiserator/pkg/synchronizer"
	pov1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_log "sigs.k8s.io/controller-runtime/pkg/log"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
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

	if cfg.Features.Vault {
		err = cfg.Vault.Validate()
		if err != nil {
			return err
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
	logSink := &logrus2logr.Logrus2Logr{Logger: log.StandardLogger()}
	ctrl_log.SetLogger(logr.New(logSink))
	mgr, err := ctrl.NewManager(kconfig, ctrl.Options{
		Cache: cache.Options{
			SyncPeriod: &cfg.Informer.FullSyncInterval,
		},
		Scheme: kscheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.Bind,
		},
		HealthProbeBindAddress: cfg.HealthProbeBindAddress,
		Logger:                 logr.New(logSink),
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
	if cfg.Features.GCP {
		listers = append(listers, naiserator_scheme.GCPListers()...)

		if len(cfg.AivenProject) > 0 {
			listers = append(listers, naiserator_scheme.AivenListers()...)
		}
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
		listers,
		kscheme,
	))

	err = naisjobReconciler.SetupWithManager(mgr, opts...)
	if err != nil {
		return err
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}
