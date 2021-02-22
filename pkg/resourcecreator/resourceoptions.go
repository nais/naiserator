package resourcecreator

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// ResourceOptions defines customizations for resource objects.
type ResourceOptions struct {
	NumReplicas                      int32
	AccessPolicy                     bool
	AccessPolicyNotAllowedCIDRs      []string
	NativeSecrets                    bool
	GoogleProjectId                  string
	GoogleTeamProjectId              string
	ApiServerIp                      string
	ClusterName                      string
	JwkerEnabled                     bool
	JwkerSecretName                  string
	JwkerServiceEntryHosts           []string
	AzureratorEnabled                bool
	AzureratorSecretName             string
	AzureratorServiceEntryHosts      []string
	KafkaratorEnabled                bool
	KafkaratorSecretName             string
	DigdiratorEnabled                bool
	DigdiratorIDPortenSecretName     string
	DigdiratorMaskinportenSecretName string
	DigdiratorServiceEntryHosts      []string
	HostAliases                      []config.HostAlias
	GatewayMappings                  []config.GatewayMapping
	VirtualServiceRegistryEnabled    bool
}

// NewResourceOptions creates a struct with the default resource options.
func NewResourceOptions() ResourceOptions {
	return ResourceOptions{
		NumReplicas: 1,
	}
}
