package main

import (
	"os"
	"time"

	azure_microsoft_com_v1alpha1 "github.com/nais/liberator/pkg/apis/azure.microsoft.com/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	security_istio_io_v1beta1 "github.com/nais/liberator/pkg/apis/security.istio.io/v1beta1"
	naiserator_scheme "github.com/nais/naiserator/pkg/scheme"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/nais/naiserator/pkg/controllers"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"

	liberator_scheme "github.com/nais/liberator/pkg/scheme"

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

	log.Info("Nebula-naiserator shutting down")
}

func run() error {
	var err error

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)
	log.SetLevel(log.DebugLevel)

	log.Info("Nebula-naiserator starting up")

	cfg, err := config.New()
	if err != nil {
		return err
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
	resourceOptions.AzureSubscriptionName = cfg.Azure.SubscriptionName
	resourceOptions.AzureSubscriptionId = cfg.Azure.SubscriptionId
	resourceOptions.AzureDomainName = cfg.Azure.DomainName

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

	listers := naiserator_scheme.GenericListers()
	listers = append(listers, IstioListers()...)
	listers = append(listers, ASOListers()...)

	skatteetatenApplicationReconciler := controllers.NewSkatteetatenAppReconciler(synchronizer.Synchronizer{
		Client:          mgrClient,
		Config:          *cfg,
		Kafka:           nil,
		ResourceOptions: resourceOptions,
		RolloutMonitor:  make(map[client.ObjectKey]synchronizer.RolloutMonitor),
		Scheme:          kscheme,
		Listers:         listers,
		SimpleClient:    simpleClient,
	})

	if err = skatteetatenApplicationReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	return mgr.Start(ctrl.SetupSignalHandler())
}


func ASOListers() []client.ObjectList {
	return []client.ObjectList{
		&azure_microsoft_com_v1alpha1.PostgreSQLDatabaseList{},
		&azure_microsoft_com_v1alpha1.PostgreSQLUserList{},
	}
}

func IstioListers() [] client.ObjectList {
	return []client.ObjectList{
		&security_istio_io_v1beta1.AuthorizationPolicyList{},
		&networking_istio_io_v1alpha3.ServiceEntryList{},
		&networking_istio_io_v1alpha3.VirtualServiceList{},
	}
}
