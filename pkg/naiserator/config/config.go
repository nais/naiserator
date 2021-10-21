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

type Features struct {
	Linkerd                     bool     `json:"linkerd"`
	AccessPolicyNotAllowedCIDRs []string `json:"access-policy-not-allowed-cidrs"`
	NativeSecrets               bool     `json:"native-secrets"`
	NetworkPolicy               bool     `json:"network-policy"`
	Vault                       bool     `json:"vault"`
	Jwker                       bool     `json:"jwker"`
	CNRM                        bool     `json:"cnrm"`
	Azurerator                  bool     `json:"azurerator"`
	Kafkarator                  bool     `json:"kafkarator"`
	Digdirator                  bool     `json:"digdirator"`
	GCP                         bool     `json:"gcp"`
	Webhook                     bool     `json:"webhook"`
	SecurePodSecurityContext    bool     `json:"secure-pod-security-context"`
}

type Securelogs struct {
	FluentdImage         string `json:"fluentd-image"`
	ConfigMapReloadImage string `json:"configmap-reload-image"`
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

type ServiceHosts struct {
	Azurerator []string `json:"azurerator"`
	Digdirator []string `json:"digdirator"`
	Jwker      []string `json:"jwker"`
}

type Wonderwall struct {
	Image string `json:"image"`
}

type Azure struct {
	SubscriptionName   string  `json:"subscription-name"`
	SubscriptionId     string  `json:"subscription-id"`
	DomainName         string  `json:"domain-name"`
}

type Config struct {
	DryRun                            bool             `json:"dry-run"`
	Bind                              string           `json:"bind"`
	Informer                          Informer         `json:"informer"`
	Synchronizer                      Synchronizer     `json:"synchronizer"`
	Kubeconfig                        string           `json:"kubeconfig"`
	ClusterName                       string           `json:"cluster-name"`
	GoogleProjectId                   string           `json:"google-project-id"`
	GoogleCloudSQLProxyContainerImage string           `json:"google-cloud-sql-proxy-container-image"`
	ApiServerIp                       string           `json:"api-server-ip"`
	Ratelimit                         Ratelimit        `json:"ratelimit"`
	Log                               Log              `json:"log"`
	Features                          Features         `json:"features"`
	Securelogs                        Securelogs       `json:"securelogs"`
	Proxy                             Proxy            `json:"proxy"`
	Vault                             Vault            `json:"vault"`
	Kafka                             Kafka            `json:"kafka"`
	HostAliases                       []HostAlias      `json:"host-aliases"`
	GatewayMappings                   []GatewayMapping `json:"gateway-mappings"`
	ServiceHosts                      ServiceHosts     `json:"service-hosts"`
	Wonderwall                        Wonderwall       `json:"wonderwall"`
	Azure                             Azure            `json:"azure"`
}

const (
	ApiServerIp                         = "api-server-ip"
	Bind                                = "bind"
	ClusterName                         = "cluster-name"
	DryRun                              = "dry-run"
	FeaturesAccessPolicyNotAllowedCIDRs = "features.access-policy-not-allowed-cidrs"
	FeaturesAzurerator                  = "features.azurerator"
	FeaturesDigdirator                  = "features.digdirator"
	FeaturesGCP                         = "features.gcp"
	FeaturesJwker                       = "features.jwker"
	FeaturesCNRM                        = "features.cnrm"
	FeaturesKafkarator                  = "features.kafkarator"
	FeaturesLinkerd                     = "features.linkerd"
	FeaturesNativeSecrets               = "features.native-secrets"
	FeaturesNetworkPolicy               = "features.network-policy"
	FeaturesSecurePodSecurityContext    = "features.secure-pod-security-context"
	FeaturesVault                       = "features.vault"
	FeaturesWebhook                     = "features.webhook"
	GoogleCloudSQLProxyContainerImage   = "google-cloud-sql-proxy-container-image"
	GoogleProjectId                     = "google-project-id"
	InformerFullSynchronizationInterval = "informer.full-sync-interval"
	KafkaBrokers                        = "kafka.brokers"
	KafkaEnabled                        = "kafka.enabled"
	KafkaLogVerbosity                   = "kafka.log-verbosity"
	KafkaTLSCAPath                      = "kafka.tls.ca-path"
	KafkaTLSCertificatePath             = "kafka.tls.certificate-path"
	KafkaTLSEnabled                     = "kafka.tls.enabled"
	KafkaTLSInsecure                    = "kafka.tls.insecure"
	KafkaTLSPrivateKeyPath              = "kafka.tls.private-key-path"
	KafkaTopic                          = "kafka.topic"
	KubeConfig                          = "kubeconfig"
	ProxyAddress                        = "proxy.address"
	ProxyExclude                        = "proxy.exclude"
	RateLimitBurst                      = "ratelimit.burst"
	RateLimitQPS                        = "ratelimit.qps"
	SecurelogsConfigMapReloadImage      = "securelogs.configmap-reload-image"
	SecurelogsFluentdImage              = "securelogs.fluentd-image"
	ServiceHostsAzurerator              = "service-hosts.azurerator"
	ServiceHostsDigdirator              = "service-hosts.digdirator"
	ServiceHostsJwker                   = "service-hosts.jwker"
	SynchronizerRolloutCheckInterval    = "synchronizer.rollout-check-interval"
	SynchronizerRolloutTimeout          = "synchronizer.rollout-timeout"
	SynchronizerSynchronizationTimeout  = "synchronizer.synchronization-timeout"
	VaultAddress                        = "vault.address"
	VaultAuthPath                       = "vault.auth-path"
	VaultInitContainerImage             = "vault.init-container-image"
	VaultKvPath                         = "vault.kv-path"
	WonderwallImage                     = "wonderwall.image"
	AzureSubscriptionName               = "azure.subscription-name"
	AzureSubscriptionId                 = "azure.subscription-id"
	AzureDomainName                     = "azure.domain-name"
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
	flag.String(ClusterName, "cluster-name-unconfigured", "cluster name as presented to deployed applications")
	flag.String(GoogleProjectId, "", "GCP project-id to store google service accounts")
	flag.String(GoogleCloudSQLProxyContainerImage, "", "Docker image of Cloud SQL Proxy container")
	flag.String(ApiServerIp, "", "IP to master in GCP, e.g. 172.16.0.2/32 for GCP")
	flag.Bool(FeaturesLinkerd, false, "enable creation of Linkerd-specific resources")
	flag.StringSlice(
		FeaturesAccessPolicyNotAllowedCIDRs, []string{""},
		"CIDRs that should not be included within the allowed IP Block rule for network policy",
	)
	flag.Bool(FeaturesNativeSecrets, false, "enable use of native secrets")
	flag.Bool(FeaturesNetworkPolicy, false, "enable creation of network policies")
	flag.Bool(FeaturesVault, false, "enable use of vault secret injection")
	flag.Bool(FeaturesGCP, false, "running in gcp and enable use of CNRM resources")
	flag.Bool(FeaturesJwker, false, "enable creation of Jwker resources and secret injection")
	flag.Bool(FeaturesCNRM, false, "enable creation of CNRM resources")
	flag.Bool(FeaturesAzurerator, false, "enable creation of AzureAdApplication resources and secret injection")
	flag.Bool(FeaturesKafkarator, false, "enable Kafkarator secret injection")
	flag.Bool(FeaturesDigdirator, false, "enable creation of IDPorten client resources and secret injection")
	flag.Bool(FeaturesWebhook, false, "enable admission webhook server")
	flag.Bool(FeaturesSecurePodSecurityContext, false, "enforce restrictive pod security context")

	flag.StringSlice(
		ServiceHostsAzurerator, []string{}, "list of hosts to output to ServiceEntry for Applications using Azurerator",
	)
	flag.StringSlice(
		ServiceHostsDigdirator, []string{}, "list of hosts to output to ServiceEntry for Applications using Digdirator",
	)
	flag.StringSlice(
		ServiceHostsJwker, []string{}, "list of hosts to output to ServiceEntry for Applications using Jwker",
	)

	flag.Duration(
		InformerFullSynchronizationInterval, time.Duration(30*time.Minute),
		"how often to run a full synchronization of all applications",
	)

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

	flag.String(SecurelogsFluentdImage, "", "Docker image used for secure log fluentd sidecar")
	flag.String(SecurelogsConfigMapReloadImage, "", "Docker image used for secure log configmap reload sidecar")

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

	flag.String(AzureSubscriptionName, "", "Name of subscription")
	flag.String(AzureSubscriptionId, "", "UUID of subscription")
	flag.String(AzureDomainName, "", "The postfix of the domain")
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
