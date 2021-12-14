package generators

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// Options defines customizations for resource objects.
// FIXME: clean up
type Options struct {
	AccessPolicyNotAllowedCIDRs       []string
	ApiServerIp                       string
	AzureratorEnabled                 bool
	Config                            config.Config
	DigdiratorEnabled                 bool
	DigdiratorHosts                   []string
	GoogleCloudSQLProxyContainerImage string
	GoogleProjectID                   string
	GoogleTeamProjectID               string
	JwkerEnabled                      bool
	SecurePodSecurityContext          bool
	AllowedKernelCapabilities         []string
	CNRMEnabled                       bool
	NetworkPolicy                     bool
	KafkaratorEnabled                 bool
	Linkerd                           bool
	NativeSecrets                     bool
	NumReplicas                       int32
	Team                              string
	VaultEnabled                      bool
	WonderwallEnabled                 bool
}

func (o *Options) IsLinkerdEnabled() bool {
	return o.Linkerd
}

func (o *Options) GetAPIServerIP() string {
	return o.ApiServerIp
}

func (o *Options) GetAccessPolicyNotAllowedCIDRs() []string {
	return o.AccessPolicyNotAllowedCIDRs
}

func (o *Options) GetClusterName() string {
	return o.Config.ClusterName
}

func (o *Options) GetGatewayMappings() []config.GatewayMapping {
	return o.Config.GatewayMappings
}

func (o *Options) IsNetworkPolicyEnabled() bool {
	return o.Config.Features.NetworkPolicy
}

func (o *Options) GetTeam() string {
	return o.Team
}

func (o *Options) IsDigdiratorEnabled() bool {
	return o.DigdiratorEnabled
}

func (o *Options) IsAzureratorEnabled() bool {
	return o.AzureratorEnabled
}

func (o *Options) IsWonderwallEnabled() bool {
	return o.WonderwallEnabled
}

func (o *Options) GetGoogleProjectID() string {
	return o.GoogleProjectID
}

func (o *Options) GetWonderwallImage() string {
	return o.Config.Wonderwall.Image
}
