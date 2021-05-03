package resourceutils

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// Options defines customizations for resource objects.
type Options struct {
	AccessPolicyNotAllowedCIDRs      []string
	ApiServerIp                      string
	AzureratorEnabled                bool
	AzureratorSecretName             string
	AzureratorServiceEntryHosts      []string
	ClusterName                      string
	DigdiratorEnabled                bool
	DigdiratorIDPortenSecretName     string
	DigdiratorMaskinportenSecretName string
	DigdiratorServiceEntryHosts      []string
	GatewayMappings                  []config.GatewayMapping
	GoogleProjectId                  string
	GoogleTeamProjectId              string
	HostAliases                      []config.HostAlias
	JwkerEnabled                     bool
	JwkerSecretName                  string
	JwkerServiceEntryHosts           []string
	NetworkPolicy                    bool
	KafkaratorEnabled                bool
	KafkaratorSecretName             string
	Linkerd                          bool
	NativeSecrets                    bool
	NumReplicas                      int32
}

// NewResourceOptions creates a struct with the default resource options.
func NewResourceOptions() Options {
	return Options{
		NumReplicas: 1,
	}
}
