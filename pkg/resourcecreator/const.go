package resourcecreator

const (
	PrometheusPodSelectorLabelValue = "prometheus" // Label value denoting the promethues pod-selector
	PrometheusNamespace             = "nais"       // Which namespace Prometheus is installed in (todo: config?)
	NginxNamespace                  = "nginx"      // Which namespace Nginx ingress controller runs in (todo: config?)

	NetworkPolicyDefaultEgressAllowIPBlock = "0.0.0.0/0" // The default IP block CIDR for the default allow network policies per app
)
