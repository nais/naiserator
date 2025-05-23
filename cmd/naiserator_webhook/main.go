package main

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/nais/liberator/pkg/logrus2logr"
	liberator_scheme "github.com/nais/liberator/pkg/scheme"
	"github.com/nais/naiserator/pkg/metrics"
	"github.com/nais/naiserator/pkg/naiserator/config"
	log "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrl_log "sigs.k8s.io/controller-runtime/pkg/log"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	err := run()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Info("Naiserator webhook shutting down")
}

func run() error {
	var err error

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)
	log.SetLevel(log.DebugLevel)

	log.Info("Naiserator webhook starting up")

	cfg, err := config.New()
	if err != nil {
		return err
	}

	config.Print([]string{
		"kafka.sasl.username",
		"kafka.sasl.password",
	})

	// Register CRDs with controller-tools
	kscheme, err := liberator_scheme.All()
	if err != nil {
		return err
	}

	logSink := &logrus2logr.Logrus2Logr{Logger: log.StandardLogger()}
	ctrl_log.SetLogger(logr.New(logSink.WithName("controller-runtime")))

	kconfig, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	kconfig.QPS = float32(cfg.Ratelimit.QPS)
	kconfig.Burst = cfg.Ratelimit.Burst

	metrics.Register(kubemetrics.Registry)
	mgr, err := ctrl.NewManager(kconfig, ctrl.Options{
		Scheme: kscheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.Bind,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Host: "0.0.0.0",
			Port: 8443,
		}),
		Logger: logr.New(logSink.WithName("controller")),
	})
	if err != nil {
		return err
	}

	// Register create/update validation webhooks for liberator_scheme's CRDs
	if err := liberator_scheme.Webhooks(mgr); err != nil {
		return err
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}
