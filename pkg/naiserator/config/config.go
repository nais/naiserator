package config

import (
	"sort"
	"strings"

	"github.com/nais/naiserator/pkg/kafka"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type Log struct {
	Format string `json:"format"`
	Level  string `json:"level"`
}

type Features struct {
	AccessPolicy  bool `json:"access-policy"`
	NativeSecrets bool `json:"native-secrets"`
	Vault         bool `json:"vault"`
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

type Config struct {
	Bind        string       `json:"bind"`
	Kubeconfig  string       `json:"kubeconfig"`
	ClusterName string       `json:"cluster-name"`
	Log         Log          `json:"log"`
	Features    Features     `json:"features"`
	Securelogs  Securelogs   `json:"securelogs"`
	Proxy       Proxy        `json:"proxy"`
	Vault       Vault        `json:"vault"`
	Kafka       kafka.Config `json:"kafka"`
}

var (
	// These configuration options should never be printed.
	redactKeys = []string{}
)

const (
	Bind                           = "bind"
	ClusterName                    = "cluster-name"
	FeaturesAccessPolicy           = "features.access-policy"
	FeaturesNativeSecrets          = "features.native-secrets"
	FeaturesVault                  = "features.vault"
	KubeConfig                     = "kubeconfig"
	ProxyAddress                   = "proxy.address"
	ProxyExclude                   = "proxy.exclude"
	SecurelogsConfigMapReloadImage = "securelogs.configmap-reload-image"
	SecurelogsFluentdImage         = "securelogs.fluentd-image"
	VaultAddress                   = "vault.address"
	VaultAuthPath                  = "vault.auth-path"
	VaultInitContainerImage        = "vault.init-container-image"
	VaultKvPath                    = "vault.kv-path"
)

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

	// Provide command-line flags
	flag.String(KubeConfig, "", "path to Kubernetes config file")
	flag.String(Bind, "127.0.0.1:8080", "ip:port where http requests are served")
	flag.String(ClusterName, "cluster-name-unconfigured", "cluster name as presented to deployed applications")

	flag.Bool(FeaturesAccessPolicy, false, "enable access policy with Istio and NetworkPolicies")
	flag.Bool(FeaturesNativeSecrets, false, "enable use of native secrets")
	flag.Bool(FeaturesVault, false, "enable use of vault secret injection")

	flag.String(SecurelogsFluentdImage, "", "Docker image used for secure log fluentd sidecar")
	flag.String(SecurelogsConfigMapReloadImage, "", "Docker image used for secure log configmap reload sidecar")

	flag.String(ProxyAddress, "", "HTTPS?_PROXY environment variable injected into containers")
	flag.StringSlice(ProxyExclude, []string{"localhost"}, "list of hosts or domains injected into NO_PROXY environment variable")

	flag.String(VaultAddress, "", "address of the Vault server")
	flag.String(VaultInitContainerImage, "", "Docker image of init container to use to read secrets from Vault")
	flag.String(VaultAuthPath, "", "path to vault kubernetes auth backend")
	flag.String(VaultKvPath, "", "path to Vault KV mount")

	kafka.SetupFlags()
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
			log.Printf("%s: %s", key, viper.GetString(key))
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
