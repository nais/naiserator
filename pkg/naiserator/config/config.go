package config

import (
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type KafkaTLS struct {
	CAPath          string `json:"ca-path"`
	CertificatePath string `json:"certificate-path"`
	Enabled         bool   `json:"enabled"`
	Insecure        bool   `json:"insecure"`
	PrivateKeyPath  string `json:"private-key-path"`
}

type Kafka struct {
	Brokers      []string `json:"brokers"`
	Enabled      bool     `json:"enabled"`
	LogVerbosity string   `json:"log-verbosity"`
	TLS          KafkaTLS `json:"tls"`
	Topic        string   `json:"topic"`
}

type Texas struct {
	Image string `json:"image"`
}

type Log struct {
	Format string `json:"format"`
	Level  string `json:"level"`
}

type Informer struct {
	FullSyncInterval time.Duration `json:"full-sync-interval"`
}

type Synchronizer struct {
	SynchronizationTimeout time.Duration `json:"synchronization-timeout"`
	RolloutTimeout         time.Duration `json:"rollout-timeout"`
	RolloutCheckInterval   time.Duration `json:"rollout-check-interval"`
}

// Keep this list sorted!
type Features struct {
	AccessPolicyNotAllowedCIDRs []string `json:"access-policy-not-allowed-cidrs"`
	Azurerator                  bool     `json:"azurerator"`
	CNRM                        bool     `json:"cnrm"`
	GARToleration               bool     `json:"gar-toleration"`
	GCP                         bool     `json:"gcp"`
	IDPorten                    bool     `json:"idporten"`
	InfluxCredentials           bool     `json:"influx-credentials"`
	Jwker                       bool     `json:"jwker"`
	Kafkarator                  bool     `json:"kafkarator"`
	Maskinporten                bool     `json:"maskinporten"`
	NAVCABundle                 bool     `json:"nav-ca-bundle"`
	NetworkPolicy               bool     `json:"network-policy"`
	PrometheusOperator          bool     `json:"prometheus-operator"`
	SqlInstanceInSharedVpc      bool     `json:"sql-instance-in-shared-vpc"`
	Texas                       bool     `json:"texas"`
	Vault                       bool     `json:"vault"`
	Webhook                     bool     `json:"webhook"`
	Wonderwall                  bool     `json:"wonderwall"`
}

type Observability struct {
	Logging Logging `json:"logging"`
	Otel    Otel    `json:"otel"`
}

type Otel struct {
	Enabled             bool                `json:"enabled"`
	Collector           OtelCollector       `json:"collector"`
	AutoInstrumentation AutoInstrumentation `json:"auto-instrumentation"`
	Destinations        []string            `json:"destinations"`
}

type AutoInstrumentation struct {
	Enabled   bool   `json:"enabled"`
	AppConfig string `json:"app-config"`
}

type OtelCollector struct {
	Labels    []string `json:"labels"`
	Namespace string   `json:"namespace"`
	Port      int      `json:"port"`
	Protocol  string   `json:"protocol"`
	Service   string   `json:"service"`
	Tls       bool     `json:"tls"`
}

type Logging struct {
	Destinations []string `json:"destinations"`
}

type Securelogs struct {
	LogShipperImage string `json:"log-shipper-image"`
}

type Proxy struct {
	Address string   `json:"address"`
	Exclude []string `json:"exclude"`
}

type Vault struct {
	Address            string `json:"address"`
	InitContainerImage string `json:"init-container-image"`
	AuthPath           string `json:"auth-path"`
	KeyValuePath       string `json:"kv-path"`
}

type GatewayMapping struct {
	DomainSuffix string `json:"domainSuffix"`
	IngressClass string `json:"ingressClass"` // Nginx
}

type HostAlias struct {
	Host    string `json:"host"`
	Address string `json:"address"`
}

type Ratelimit struct {
	QPS   int `json:"qps"`
	Burst int `json:"burst"`
}

type Wonderwall struct {
	Image string `json:"image"`
}

type LeaderElection struct {
	Image string `json:"image"`
}

type FQDNPolicy struct {
	Enabled bool       `json:"enabled"`
	Rules   []FQDNRule `json:"rules"`
}

type FQDNRule struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Frontend struct {
	TelemetryURL string `json:"telemetry-url"`
}

type Config struct {
	AivenGeneration                   int              `json:"aiven-generation"`
	AivenProject                      string           `json:"aiven-project"`
	AivenRange                        string           `json:"aiven-range"`
	ApiServerIp                       string           `json:"api-server-ip"`
	Bind                              string           `json:"bind"`
	ClusterName                       string           `json:"cluster-name"`
	DocUrl                            string           `json:"doc-url"`
	DryRun                            bool             `json:"dry-run"`
	FQDNPolicy                        FQDNPolicy       `json:"fqdn-policy"`
	Features                          Features         `json:"features"`
	Frontend                          Frontend         `json:"frontend"`
	GatewayMappings                   []GatewayMapping `json:"gateway-mappings"`
	GoogleCloudSQLProxyContainerImage string           `json:"google-cloud-sql-proxy-container-image"`
	GoogleProjectId                   string           `json:"google-project-id"`
	HealthProbeBindAddress            string           `json:"health-probe-bind-address"`
	HostAliases                       []HostAlias      `json:"host-aliases"`
	ImagePullSecrets                  []string         `json:"image-pull-secrets"`
	Informer                          Informer         `json:"informer"`
	Kafka                             Kafka            `json:"kafka"`
	Kubeconfig                        string           `json:"kubeconfig"`
	LeaderElection                    LeaderElection   `json:"leader-election"`
	Log                               Log              `json:"log"`
	MaxConcurrentReconciles           int              `json:"max-concurrent-reconciles"`
	NaisNamespace                     string           `json:"nais-namespace"`
	Observability                     Observability    `json:"observability"`
	Proxy                             Proxy            `json:"proxy"`
	Ratelimit                         Ratelimit        `json:"ratelimit"`
	Securelogs                        Securelogs       `json:"securelogs"`
	Synchronizer                      Synchronizer     `json:"synchronizer"`
	Texas                             Texas            `json:"texas"`
	Vault                             Vault            `json:"vault"`
	Wonderwall                        Wonderwall       `json:"wonderwall"`
}

const (
	AivenGeneration                               = "aiven-generation"
	AivenProject                                  = "aiven-project"
	AivenRange                                    = "aiven-range"
	ApiServerIp                                   = "api-server-ip"
	Bind                                          = "bind"
	HealthProbeBindAddress                        = "health-probe-bind-address"
	ClusterName                                   = "cluster-name"
	DryRun                                        = "dry-run"
	NaisNamespace                                 = "nais-namespace"
	FeaturesAccessPolicyNotAllowedCIDRs           = "features.access-policy-not-allowed-cidrs"
	FeaturesAzurerator                            = "features.azurerator"
	FeaturesGCP                                   = "features.gcp"
	FeaturesIDPorten                              = "features.idporten"
	FeaturesJwker                                 = "features.jwker"
	FeaturesCNRM                                  = "features.cnrm"
	FeaturesKafkarator                            = "features.kafkarator"
	FeaturesMaskinporten                          = "features.maskinporten"
	FeaturesNetworkPolicy                         = "features.network-policy"
	FeaturesPrometheusOperator                    = "features.prometheus-operator"
	FeaturesTexas                                 = "features.texas"
	FeaturesVault                                 = "features.vault"
	FeaturesWebhook                               = "features.webhook"
	FeaturesWonderwall                            = "features.wonderwall"
	FQDNPolicyEnabled                             = "fqdn-policy.enabled"
	GoogleCloudSQLProxyContainerImage             = "google-cloud-sql-proxy-container-image"
	GoogleProjectId                               = "google-project-id"
	ImagePullSecrets                              = "image-pull-secrets"
	InformerFullSynchronizationInterval           = "informer.full-sync-interval"
	KafkaBrokers                                  = "kafka.brokers"
	KafkaEnabled                                  = "kafka.enabled"
	KafkaLogVerbosity                             = "kafka.log-verbosity"
	KafkaTLSCAPath                                = "kafka.tls.ca-path"
	KafkaTLSCertificatePath                       = "kafka.tls.certificate-path"
	KafkaTLSEnabled                               = "kafka.tls.enabled"
	KafkaTLSInsecure                              = "kafka.tls.insecure"
	KafkaTLSPrivateKeyPath                        = "kafka.tls.private-key-path"
	KafkaTopic                                    = "kafka.topic"
	KubeConfig                                    = "kubeconfig"
	LeaderElectionImage                           = "leader-election.image"
	MaxConcurrentReconciles                       = "max-concurrent-reconciles"
	ObservabilityLoggingDestinations              = "observability.logging.destinations"
	ObservabilityOtelCollectorLabels              = "observability.otel.collector.labels"
	ObservabilityOtelCollectorNamespace           = "observability.otel.collector.namespace"
	ObservabilityOtelCollectorPort                = "observability.otel.collector.port"
	ObservabilityOtelCollectorProtocol            = "observability.otel.collector.protocol"
	ObservabilityOtelCollectorService             = "observability.otel.collector.service"
	ObservabilityOtelCollectorTLS                 = "observability.otel.collector.tls"
	ObservabilityOtelEnabled                      = "observability.otel.enabled"
	ObservabilityOtelDestinations                 = "observability.otel.destinations"
	ObservabilityOtelAutoInstrumentationAppConfig = "observability.otel.auto-instrumentation.app-config"
	ObservabilityOtelAutoInstrumentationEnabled   = "observability.otel.auto-instrumentation.enabled"
	ProxyAddress                                  = "proxy.address"
	ProxyExclude                                  = "proxy.exclude"
	RateLimitBurst                                = "ratelimit.burst"
	RateLimitQPS                                  = "ratelimit.qps"
	SecurelogsLogShipperImage                     = "securelogs.log-shipper-image"
	SynchronizerRolloutCheckInterval              = "synchronizer.rollout-check-interval"
	SynchronizerRolloutTimeout                    = "synchronizer.rollout-timeout"
	SynchronizerSynchronizationTimeout            = "synchronizer.synchronization-timeout"
	TexasImage                                    = "texas.image"
	VaultAddress                                  = "vault.address"
	VaultAuthPath                                 = "vault.auth-path"
	VaultInitContainerImage                       = "vault.init-container-image"
	VaultKvPath                                   = "vault.kv-path"
	WonderwallImage                               = "wonderwall.image"
)

func bindNAIS() {
	viper.BindEnv(KafkaBrokers, "KAFKA_BROKERS")
	viper.BindEnv(KafkaTLSCAPath, "KAFKA_CA_PATH")
	viper.BindEnv(KafkaTLSCertificatePath, "KAFKA_CERTIFICATE_PATH")
	viper.BindEnv(KafkaTLSPrivateKeyPath, "KAFKA_PRIVATE_KEY_PATH")
}

func init() {
	// Automatically read configuration options from environment variables.
	// i.e. --proxy.address will be configurable using NAISERATOR_PROXY_ADDRESS.
	viper.SetEnvPrefix("NAISERATOR")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Read configuration file from working directory and/or /etc.
	// File formats supported include JSON, TOML, YAML, HCL, envfile and Java properties config files
	viper.SetConfigName("naiserator")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")

	// Ensure Kafkarator variables are used
	bindNAIS()

	// Provide command-line flags
	flag.Bool(DryRun, false, "set to true to run without any actual changes to the cluster")
	flag.String(KubeConfig, "", "path to Kubernetes config file")
	flag.String(Bind, "127.0.0.1:8080", "ip:port where http requests are served")
	flag.String(HealthProbeBindAddress, "127.0.0.1:8085", "ip:port where health probes are performed")
	flag.String(ClusterName, "cluster-name-unconfigured", "cluster name as presented to deployed applications")
	flag.Int(AivenGeneration, 0, "the generation of aiven secrets in this cluster")
	flag.String(AivenProject, "aiven-project", "main Aiven project for this cluster")
	flag.String(AivenRange, "aiven-range", "range of IP addresses for Aiven services")
	flag.String(GoogleProjectId, "", "GCP project-id to store google service accounts")
	flag.String(GoogleCloudSQLProxyContainerImage, "", "Docker image of Cloud SQL Proxy container")
	flag.String(ApiServerIp, "", "IP to master in GCP, e.g. 172.16.0.2/32 for GCP")
	flag.String(NaisNamespace, "nais-system", "namespace where nais resources are deployed")
	flag.StringSlice(
		FeaturesAccessPolicyNotAllowedCIDRs, []string{""},
		"CIDRs that should not be included within the allowed IP Block rule for network policy",
	)
	flag.Bool(FeaturesNetworkPolicy, false, "enable creation of network policies")
	flag.Bool(FeaturesVault, false, "enable use of vault secret injection")
	flag.Bool(FeaturesGCP, false, "running in gcp and enable use of CNRM resources")
	flag.Bool(FeaturesJwker, false, "enable creation of Jwker resources and secret injection")
	flag.Bool(FeaturesCNRM, false, "enable creation of CNRM resources")
	flag.Bool(FeaturesAzurerator, false, "enable creation of AzureAdApplication resources and secret injection")
	flag.Bool(FeaturesKafkarator, false, "enable Kafkarator secret injection")
	flag.Bool(FeaturesIDPorten, false, "enable creation of IDPorten client resources and secret injection")
	flag.Bool(FeaturesMaskinporten, false, "enable creation of Maskinporten client resources and secret injection")
	flag.Bool(FeaturesWebhook, false, "enable admission webhook server")
	flag.Bool(FeaturesPrometheusOperator, false, "enable Prometheus Operator")
	flag.Bool(FeaturesWonderwall, false, "enable Wonderwall sidecar")
	flag.Bool(FeaturesTexas, false, "enable token exchange as a sidecar/service")
	flag.Bool(FQDNPolicyEnabled, false, "enable FQDN policies")
	flag.Duration(
		InformerFullSynchronizationInterval, time.Duration(30*time.Minute),
		"how often to run a full synchronization of all applications",
	)

	flag.String(LeaderElectionImage, "", "image to use for leader election in deployed applications")
	flag.Int(MaxConcurrentReconciles, 1, "maximum number of concurrent Reconciles which can be run by the controller.")
	flag.StringArray(ObservabilityLoggingDestinations, []string{}, "list of valid logging destinations")
	flag.Bool(ObservabilityOtelEnabled, false, "enable OpenTelemetry")
	flag.StringArray(ObservabilityOtelDestinations, []string{}, "list of valid otel storage destinations")
	flag.String(ObservabilityOtelCollectorNamespace, "nais-system", "namespace of the OpenTelemetry collector")
	flag.String(ObservabilityOtelCollectorService, "opentelmetry-collector", "service name of the OpenTelemetry collector")
	flag.String(ObservabilityOtelCollectorProtocol, "grpc", "protocol used by the OpenTelemetry collector")
	flag.Bool(ObservabilityOtelAutoInstrumentationEnabled, false, "enable OpenTelemetry auto-instrumentation")
	flag.String(ObservabilityOtelAutoInstrumentationAppConfig, "nais-system/apps", "path to OpenTelemetry auto-instrumentation config")
	flag.Int(ObservabilityOtelCollectorPort, 4317, "port used by the OpenTelemetry collector")
	flag.Bool(ObservabilityOtelCollectorTLS, false, "use TLS for the OpenTelemetry collector")
	flag.StringArray(ObservabilityOtelCollectorLabels, []string{}, "list of labels to be used by the OpenTelemetry collector")
	flag.Int(RateLimitQPS, 20, "how quickly the rate limit burst bucket is filled per second")
	flag.Int(RateLimitBurst, 200, "how many requests to Kubernetes to allow per second")

	flag.Duration(
		SynchronizerSynchronizationTimeout, time.Duration(5*time.Second),
		"how long to allow for resource creation on a single application",
	)
	flag.Duration(
		SynchronizerRolloutCheckInterval, time.Duration(5*time.Second),
		"how often to check if a deployment has rolled out successfully",
	)
	flag.Duration(
		SynchronizerRolloutTimeout, time.Duration(5*time.Minute),
		"how long to keep checking for a successful deployment rollout",
	)

	flag.String(SecurelogsLogShipperImage, "", "Docker image used for shipping secure logs")

	flag.String(TexasImage, "", "Docker image used for Texas")

	flag.String(ProxyAddress, "", "HTTPS?_PROXY environment variable injected into containers")
	flag.StringSlice(
		ProxyExclude, []string{"localhost"}, "list of hosts or domains injected into NO_PROXY environment variable",
	)

	flag.String(VaultAddress, "", "address of the Vault server")
	flag.String(VaultInitContainerImage, "", "Docker image of init container to use to read secrets from Vault")
	flag.String(VaultAuthPath, "", "path to vault kubernetes auth backend")
	flag.String(VaultKvPath, "", "path to Vault KV mount")

	flag.Bool(KafkaEnabled, false, "Enable connection to kafka")
	flag.Bool(KafkaTLSEnabled, false, "Use TLS for connecting to Kafka.")
	flag.Bool(KafkaTLSInsecure, false, "Allow insecure Kafka TLS connections.")
	flag.String(KafkaLogVerbosity, "trace", "Log verbosity for Kafka client.")
	flag.String(KafkaTLSCAPath, "", "Path to Kafka TLS CA certificate.")
	flag.String(KafkaTLSCertificatePath, "", "Path to Kafka TLS certificate.")
	flag.String(KafkaTLSPrivateKeyPath, "", "Path to Kafka TLS private key.")
	flag.String(KafkaTopic, "deploymentEvents", "Kafka topic for deployment status.")
	flag.StringSlice(KafkaBrokers, []string{"localhost:9092"}, "Comma-separated list of Kafka brokers, HOST:PORT.")

	flag.String(WonderwallImage, "", "Docker image used for Wonderwall.")

	flag.StringArray(ImagePullSecrets, nil, "List of image pull secrets to use for pulling images")
}

// Print out all configuration options except secret stuff.
func Print(redacted []string) {
	ok := func(key string) bool {
		for _, forbiddenKey := range redacted {
			if forbiddenKey == key {
				return false
			}
		}
		return true
	}

	var keys sort.StringSlice = viper.AllKeys()

	keys.Sort()
	for _, key := range keys {
		if ok(key) {
			log.Printf("%s: %v", key, viper.Get(key))
		} else {
			log.Printf("%s: ***REDACTED***", key)
		}
	}
}

func decoderHook(dc *mapstructure.DecoderConfig) {
	dc.TagName = "json"
	dc.ErrorUnused = true
}

func New() (*Config, error) {
	var err error
	var cfg Config

	err = viper.ReadInConfig()
	if err != nil {
		if err.(viper.ConfigFileNotFoundError) != err {
			return nil, err
		}
	}

	flag.Parse()

	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&cfg, decoderHook)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
