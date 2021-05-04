package resourcecreator

const (
	NaisAppNameEnv     = "NAIS_APP_NAME"
	NaisNamespaceEnv   = "NAIS_NAMESPACE"
	NaisAppImageEnv    = "NAIS_APP_IMAGE"
	NaisClusterNameEnv = "NAIS_CLUSTER_NAME"
	NaisClientId       = "NAIS_CLIENT_ID"

	PrometheusPodSelectorLabelValue         = "prometheus"                   // Label value denoting the promethues pod-selector
	PrometheusNamespace                     = "nais"                                 // Which namespace Prometheus is installed in (todo: config?)
	NginxNamespace                          = "nginx"                                // Which namespace Nginx ingress controller runs in (todo: config?)

	NetworkPolicyDefaultEgressAllowIPBlock = "0.0.0.0/0" // The default IP block CIDR for the default allow network policies per app
)
