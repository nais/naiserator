package resourcecreator

const (
	NaisAppNameEnv     = "NAIS_APP_NAME"
	NaisNamespaceEnv   = "NAIS_NAMESPACE"
	NaisAppImageEnv    = "NAIS_APP_IMAGE"
	NaisClusterNameEnv = "NAIS_CLUSTER_NAME"
	NaisClientId       = "NAIS_CLIENT_ID"

	IstioAuthorizationPolicyVersion         = "security.istio.io/v1beta1"    // API version of the AP resource
	IstioNetworkingAPIVersion               = "networking.istio.io/v1alpha3" // API version of the Networking resources
	IstioIngressGatewayLabelValue           = "ingressgateway"               // Label value denoting the ingress gateway pod selector
	PrometheusPodSelectorLabelValue         = "prometheus"                   // Label value denoting the promethues pod-selector
	IstioPrometheusPort                     = "15090"
	IstioIngressGatewayServiceAccount       = "istio-ingressgateway-service-account" // Service account name that Istio ingress gateway is running as
	IstioNamespace                          = "istio-system"                         // Which namespace Istio is installed in
	PrometheusNamespace                     = "nais"                                 // Which namespace Prometheus is installed in (todo: config?)
	NginxNamespace                          = "nginx"                                // Which namespace Nginx ingress controller runs in (todo: config?)
	IstioServiceEntryLocationExternal       = "MESH_EXTERNAL"                        // Service entries external to the cluster
	IstioServiceEntryResolutionDNS          = "DNS"                                  // Service entry lookup type
	IstioVirtualServiceTotalWeight    int32 = 100                                    // The total weight of all routes must equal 100

	NetworkPolicyDefaultEgressAllowIPBlock = "0.0.0.0/0" // The default IP block CIDR for the default allow network policies per app
)
