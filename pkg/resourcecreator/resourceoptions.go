package resourcecreator

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// ResourceOptions defines customizations for resource objects.
type ResourceOptions struct {
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
	Istio                            bool
	JwkerEnabled                     bool
	JwkerSecretName                  string
	JwkerServiceEntryHosts           []string
	NetworkPolicy                    bool
	KafkaratorEnabled                bool
	KafkaratorSecretName             string
	Linkerd                          bool
	NativeSecrets                    bool
	NumReplicas                      int32
	VirtualServiceRegistryEnabled    bool
}

// NewResourceOptions creates a struct with the default resource options.
func NewResourceOptions() ResourceOptions {
	return ResourceOptions{
		NumReplicas: 1,
	}
}
