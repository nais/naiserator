package resourcecreator

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// ResourceOptions defines customizations for resource objects.
type ResourceOptions struct {
	NumReplicas                 int32
	AccessPolicy                bool
	AccessPolicyNotAllowedCIDRs []string
	NativeSecrets               bool
	GoogleProjectId             string
	GoogleTeamProjectId         string
	ClusterName                 string
	JwkerEnabled                bool
	JwkerSecretName             string
	AzureratorEnabled           bool
	AzureratorSecretName        string
	HostAliases                 []config.HostAlias
}

// NewResourceOptions creates a struct with the default resource options.
func NewResourceOptions() ResourceOptions {
	return ResourceOptions{
		NumReplicas: 1,
	}
}
