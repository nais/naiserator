package generator

import (
	"fmt"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"istio.io/api/security/v1beta1"
	beta1 "istio.io/api/type/v1beta1"
	v1beta12 "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"strconv"
)

const (
	IstioNamespace        = "istio-system"
	DefaultIngressGateway = "istio-ingressgateway"
	ServiceAccountSuffix  = "-service-account"
)

var appNamespace string

func GenerateAuthorizationPolicy(application skatteetaten_no_v1alpha1.Application, config skatteetaten_no_v1alpha1.ApplicationSpec) *v1beta12.AuthorizationPolicy {

	appNamespace = application.Namespace
	authPolicy := generateAuthorizationPolicy(application, v1beta1.AuthorizationPolicy_ALLOW)

	if config.Ingress == nil {
		return authPolicy
	}

	// Authorization Policies to allow ingress from configured istio gateways
	for _, ingress := range config.Ingress.Public {
		if ingress.Enabled {
			// TODO: can be removed as default is defined in application_types
			gateway := ingress.Gateway
			if len(gateway) == 0 {
				gateway = DefaultIngressGateway
			}

			authPolicy.Spec.Rules = append(
				authPolicy.Spec.Rules,

				generateAuthorizationPolicyRule(skatteetaten_no_v1alpha1.InternalIngressConfig{
					Enabled:     true,
					Application: fmt.Sprintf("%s%s", gateway, ServiceAccountSuffix),
					Namespace:   IstioNamespace,
					Ports:       []skatteetaten_no_v1alpha1.PortConfig{{Port: uint16(ingress.Port)}},
				}),
			)
		}
	}

	// Sort to allow fixture testing
	keys := make([]string, 0, len(config.Ingress.Internal))
	for k := range config.Ingress.Internal {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Authorization Policies for internal ingress
	for _, rule := range keys {
		if config.Ingress.Internal[rule].Enabled {
			authPolicy.Spec.Rules = append(
				authPolicy.Spec.Rules,
				generateAuthorizationPolicyRule(config.Ingress.Internal[rule]),
			)
		}
	}

	return authPolicy
}

func generateAuthorizationPolicyRule(rule skatteetaten_no_v1alpha1.InternalIngressConfig) *v1beta1.Rule {
	PolicyRule := &v1beta1.Rule{}

	// Namespace not set, app not set -> Allow all apps in same namespace   -> source namespace
	// Namespace set,     app not set -> Allow all apps in given namespace  -> source namespace
	// Namespace set,     app set     -> Allow given app in given namespace -> source principal
	// Namespace not set, app set     -> Allow given app in same namespace  -> source principal
	namespace := rule.Namespace
	if len(rule.Namespace) == 0 {
		namespace = appNamespace
	}

	if rule.Application == "*" || rule.Application == "" {
		PolicyRule.From = []*v1beta1.Rule_From{
			{
				Source: &v1beta1.Source{
					Namespaces: []string{namespace},
				},
			},
		}
	} else {
		PolicyRule.From = []*v1beta1.Rule_From{
			{
				Source: &v1beta1.Source{
					Principals: []string{
						fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application),
					},
				},
			},
		}
	}

	Operation := v1beta1.Operation{}

	var ports []string
	for _, port := range rule.Ports {
		ports = append(ports, strconv.Itoa(int(port.Port)))
	}

	Operation.Ports = ports
	Operation.Paths = rule.Paths
	Operation.Methods = rule.Methods

	if Operation.Size() > 0 {
		PolicyRule.To = []*v1beta1.Rule_To{{Operation: &Operation}}
	}

	return PolicyRule
}

func generateAuthorizationPolicy(application skatteetaten_no_v1alpha1.Application, action v1beta1.AuthorizationPolicy_Action) *v1beta12.AuthorizationPolicy {
	return &v1beta12.AuthorizationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.istio.io/v1beta1",
			Kind:       "AuthorizationPolicy",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: v1beta1.AuthorizationPolicy{
			Selector: &beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": application.Name},
			},
			// Requests are denied by default when no rules are defined in the policy (rules == nil) .
			// https://istio.io/latest/docs/reference/config/security/authorization-policy/#AuthorizationPolicy
			Rules:  nil,
			Action: action,
		},
	}
}
