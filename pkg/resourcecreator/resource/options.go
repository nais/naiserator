package resource

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// Options defines customizations for resource objects.
type Options struct {
	AccessPolicyNotAllowedCIDRs      []string
	ApiServerIp                      string
	AzureratorEnabled                bool
	AzureratorSecretName             string
	AzureratorHosts                  []string
	ClusterName                      string
	DigdiratorEnabled                bool
	DigdiratorIDPortenSecretName     string
	DigdiratorMaskinportenSecretName string
	DigdiratorHosts                  []string
	GatewayMappings                  []config.GatewayMapping
	GoogleProjectId                  string
	GoogleTeamProjectId              string
	HostAliases                      []config.HostAlias
	JwkerEnabled                     bool
	JwkerSecretName                  string
	JwkerHosts                       []string
	NetworkPolicy                    bool
	KafkaratorEnabled                bool
	KafkaratorSecretName             string
	Linkerd                          bool
	NativeSecrets                    bool
	NumReplicas                      int32
}

// NewOptions creates a struct with the default resource options.
func NewOptions() Options {
	return Options{
		NumReplicas: 1,
	}
}
