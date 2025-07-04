package generators

import (
	"github.com/nais/naiserator/pkg/naiserator/config"
)

type SqlInstance struct {
	exists                  bool
	hasPrivateIpInSharedVpc bool
}

// Options defines customizations for resource objects.
// These parameters are set during the Prepare() stage of the generator,
// and then passed to the different resource generators.
type Options struct {
	Config              config.Config
	GoogleProjectID     string
	GoogleTeamProjectID string
	Image               string
	NumReplicas         int32
	Team                string
	SqlInstance         SqlInstance
}

func (o *Options) GetAccessPolicyNotAllowedCIDRs() []string {
	return o.Config.Features.AccessPolicyNotAllowedCIDRs
}

func (o *Options) GetAivenGeneration() int {
	return o.Config.AivenGeneration
}

func (o *Options) GetAivenProject() string {
	return o.Config.AivenProject
}

func (o *Options) GetAivenRange() string {
	return o.Config.AivenRange
}

func (o *Options) GetAPIServerIP() string {
	return o.Config.ApiServerIp
}

func (o *Options) GetClusterName() string {
	return o.Config.ClusterName
}

func (o *Options) GetDocUrl() string {
	return o.Config.DocUrl
}

func (o *Options) GetFrontendOptions() config.Frontend {
	return o.Config.Frontend
}

func (o *Options) GetFQDNPolicy() config.FQDNPolicy {
	return o.Config.FQDNPolicy
}

func (o *Options) GetGatewayMappings() []config.GatewayMapping {
	return o.Config.GatewayMappings
}

func (o *Options) GetGoogleCloudSQLProxyContainerImage() string {
	return o.Config.GoogleCloudSQLProxyContainerImage
}

func (o *Options) GetGoogleProjectID() string {
	return o.Config.GoogleProjectId
}

func (o *Options) GetGoogleTeamProjectID() string {
	return o.GoogleTeamProjectID
}

func (o *Options) GetHostAliases() []config.HostAlias {
	return o.Config.HostAliases
}

func (o *Options) GetImagePullSecrets() []string {
	return o.Config.ImagePullSecrets
}

func (o *Options) GetLeaderElectionImage() string {
	return o.Config.LeaderElection.Image
}

func (o *Options) GetNaisNamespace() string {
	return o.Config.NaisNamespace
}

func (o *Options) GetNumReplicas() int32 {
	return o.NumReplicas
}

func (o *Options) GetObservability() config.Observability {
	return o.Config.Observability
}

func (o *Options) GetSecureLogsOptions() config.Securelogs {
	return o.Config.Securelogs
}

func (o *Options) GetTeam() string {
	return o.Team
}

func (o *Options) GetVaultOptions() config.Vault {
	return o.Config.Vault
}

func (o *Options) GetWebProxyOptions() config.Proxy {
	return o.Config.Proxy
}

func (o *Options) GetWonderwallOptions() config.Wonderwall {
	return o.Config.Wonderwall
}

func (o *Options) IsAzureratorEnabled() bool {
	return o.Config.Features.Azurerator
}

func (o *Options) IsCNRMEnabled() bool {
	return o.Config.Features.CNRM
}

func (o *Options) IsGARTolerationEnabled() bool {
	return o.Config.Features.GARToleration
}

func (o *Options) IsGCPEnabled() bool {
	return o.Config.Features.GCP
}

func (o *Options) IsIDPortenEnabled() bool {
	return o.Config.Features.IDPorten
}

func (o *Options) IsJwkerEnabled() bool {
	return o.Config.Features.Jwker
}

func (o *Options) IsKafkaratorEnabled() bool {
	return o.Config.Features.Kafkarator
}

func (o *Options) IsMaskinportenEnabled() bool {
	return o.Config.Features.Maskinporten
}

func (o *Options) IsNAVCABundleEnabled() bool {
	return o.Config.Features.NAVCABundle
}

func (o *Options) IsNetworkPolicyEnabled() bool {
	return o.Config.Features.NetworkPolicy
}

func (o *Options) IsPrometheusOperatorEnabled() bool {
	return o.Config.Features.PrometheusOperator
}

func (o *Options) IsTexasEnabled() bool {
	return o.Config.Features.Texas
}

func (o *Options) IsVaultEnabled() bool {
	return o.Config.Features.Vault
}

func (o *Options) IsWonderwallEnabled() bool {
	return o.Config.Features.Wonderwall
}

func (o *Options) PostgresImage() string {
	return o.Config.Postgres.Image
}

func (o *Options) PostgresOperatorEnabled() bool {
	return o.Config.Features.PostgresOperator
}

func (o *Options) PostgresStorageClass() string {
	return o.Config.Postgres.StorageClass
}

func (o *Options) ShouldCreateSqlInstanceInSharedVpc() bool {
	return o.Config.Features.SqlInstanceInSharedVpc
}

func (o *Options) SqlInstanceExists() bool {
	return o.SqlInstance.exists
}

func (o *Options) SqlInstanceHasPrivateIpInSharedVpc() bool {
	return o.SqlInstance.hasPrivateIpInSharedVpc
}

func (o *Options) TexasImage() string {
	return o.Config.Texas.Image
}
