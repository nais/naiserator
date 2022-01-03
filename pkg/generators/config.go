package generators

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

// Options defines customizations for resource objects.
// FIXME: clean up
type Options struct {
	AccessPolicyNotAllowedCIDRs []string
	ApiServerIp                 string
	AzureratorEnabled           bool
	Config                      config.Config
	DigdiratorEnabled           bool
	DigdiratorHosts             []string
	GoogleProjectID             string
	GoogleTeamProjectID         string
	AllowedKernelCapabilities   []string
	CNRMEnabled                 bool
	NetworkPolicy               bool
	Linkerd                     bool
	NativeSecrets               bool
	NumReplicas                 int32
	Team                        string
	VaultEnabled                bool
	WonderwallEnabled           bool
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

func (o *Options) IsCNRMEnabled() bool {
	return o.Config.Features.CNRM
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
	return o.Config.GoogleProjectId
}

func (o *Options) GetGoogleTeamProjectID() string {
	return o.GoogleTeamProjectID
}

func (o *Options) GetGoogleCloudSQLProxyContainerImage() string {
	return o.Config.GoogleCloudSQLProxyContainerImage
}

func (o *Options) GetWonderwallImage() string {
	return o.Config.Wonderwall.Image
}

func (o *Options) GetWebProxyOptions() config.Proxy {
	return o.Config.Proxy
}

func (o *Options) GetSecureLogsOptions() config.Securelogs {
	return o.Config.Securelogs
}

func (o *Options) IsJwkerEnabled() bool {
	return o.Config.Features.Jwker
}

func (o *Options) IsKafkaratorEnabled() bool {
	return o.Config.Features.Kafkarator
}

func (o *Options) IsVaultEnabled() bool {
	return o.Config.Features.Vault
}

func (o *Options) GetVaultOptions() config.Vault {
	return o.Config.Vault
}

func (o *Options) IsNativeSecretsEnabled() bool {
	return o.Config.Features.NativeSecrets
}

func (o *Options) GetHostAliases() []config.HostAlias {
	return o.Config.HostAliases
}

func (o *Options) IsSecurePodSecurityContextEnabled() bool {
	return o.Config.Features.SecurePodSecurityContext
}

func (o *Options) GetAllowedKernelCapabilities() []string {
	return []string{"NET_RAW", "NET_BIND_SERVICE"}
}

func (o *Options) GetNumReplicas() int32 {
	return o.NumReplicas
}
