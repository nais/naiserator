package resource

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// Options defines customizations for resource objects.
type Options struct {
	AccessPolicyNotAllowedCIDRs       []string
	ApiServerIp                       string
	AzureratorEnabled                 bool
	ClusterName                       string
	DigdiratorEnabled                 bool
	DigdiratorHosts                   []string
	GatewayMappings                   []config.GatewayMapping
	GoogleCloudSQLProxyContainerImage string
	GoogleProjectId                   string
	GoogleTeamProjectId               string
	HostAliases                       []config.HostAlias
	JwkerEnabled                      bool
	CNRMEnabled                       bool
	NetworkPolicy                     bool
	KafkaratorEnabled                 bool
	KafkaratorSecretName              string
	Linkerd                           bool
	NativeSecrets                     bool
	NumReplicas                       int32
	Proxy                             config.Proxy
	Securelogs                        config.Securelogs
	VaultEnabled                      bool
	Vault                             config.Vault
	Wonderwall                        config.Wonderwall
	WonderwallEnabled                 bool
	SkattUsePullSecret                bool
	AzureServiceOperatorEnabled       bool
	Istio							  bool
}

// NewOptions creates a struct with the default resource options.
func NewOptions() Options {
	return Options{
		NumReplicas: 1,
	}
}
