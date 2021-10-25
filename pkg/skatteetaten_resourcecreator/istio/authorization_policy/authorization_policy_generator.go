package authorization_policy

import (
	"fmt"
	"sort"
	"strconv"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	security_istio_io_v1beta1 "github.com/nais/liberator/pkg/apis/security.istio.io/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IstioNamespace        = "istio-system"
	DefaultIngressGateway = "istio-ingressgateway"
	ServiceAccountSuffix  = "-service-account"
)

var appNamespace string

type Source interface {
	resource.Source
	GetIngress() *skatteetaten_no_v1alpha1.IngressConfig
}

func Create(app Source, ast *resource.Ast) {
	ingressConfig := app.GetIngress()
	appNamespace = app.GetNamespace()
	authPolicy := generateAuthorizationPolicy(app, "ALLOW")

	if ingressConfig == nil {
		return
	}

	// Authorization Policies to allow ingress from configured istio gateways
	for _, ingress := range ingressConfig.Public {
		// TODO: can be removed as default is defined in application_types
		gateway := ingress.Gateway
		if len(gateway) == 0 {
			gateway = DefaultIngressGateway
		}

		authPolicy.Spec.Rules = append(
			authPolicy.Spec.Rules,

			generateAuthorizationPolicyRule(skatteetaten_no_v1alpha1.InternalIngressConfig{
				Application: fmt.Sprintf("%s%s", gateway, ServiceAccountSuffix),
				Namespace:   IstioNamespace,
				Ports:       []skatteetaten_no_v1alpha1.PortConfig{{Port: uint16(ingress.Port)}},
			}),
		)

	}

	// Sort to allow fixture testing
	keys := make([]string, 0, len(ingressConfig.Internal))
	for k := range ingressConfig.Internal {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Authorization Policies for internal ingress
	for _, rule := range keys {
		authPolicy.Spec.Rules = append(
			authPolicy.Spec.Rules,
			generateAuthorizationPolicyRule(ingressConfig.Internal[rule]),
		)
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, authPolicy)

}

func generateAuthorizationPolicyRule(rule skatteetaten_no_v1alpha1.InternalIngressConfig) *security_istio_io_v1beta1.Rule {
	PolicyRule := &security_istio_io_v1beta1.Rule{}

	// Namespace not set, app not set -> Allow all apps in same namespace   -> source namespace
	// Namespace set,     app not set -> Allow all apps in given namespace  -> source namespace
	// Namespace set,     app set     -> Allow given app in given namespace -> source principal
	// Namespace not set, app set     -> Allow given app in same namespace  -> source principal
	namespace := rule.Namespace
	if len(rule.Namespace) == 0 {
		namespace = appNamespace
	}

	if rule.Application == "*" || rule.Application == "" {
		PolicyRule.From = []*security_istio_io_v1beta1.Rule_From{
			{
				Source: &security_istio_io_v1beta1.Source{
					Namespaces: []string{namespace},
				},
			},
		}
	} else {
		PolicyRule.From = []*security_istio_io_v1beta1.Rule_From{
			{
				Source: &security_istio_io_v1beta1.Source{
					Principals: []string{
						fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application),
					},
				},
			},
		}
	}

	Operation := security_istio_io_v1beta1.Operation{}

	var ports []string
	for _, port := range rule.Ports {
		ports = append(ports, strconv.Itoa(int(port.Port)))
	}

	Operation.Ports = ports
	Operation.Paths = rule.Paths
	Operation.Methods = rule.Methods

	// TODO: need to find something to simulatte this
	// if Operation.Size() > 0 {

	PolicyRule.To = []*security_istio_io_v1beta1.Rule_To{{Operation: &Operation}}
	// }

	return PolicyRule
}

func generateAuthorizationPolicy(source resource.Source, action string) *security_istio_io_v1beta1.AuthorizationPolicy {
	return &security_istio_io_v1beta1.AuthorizationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "security.istio.io/v1beta1",
			Kind:       "AuthorizationPolicy",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: security_istio_io_v1beta1.AuthorizationPolicySpec{
			Selector: &security_istio_io_v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": source.GetName()},
			},
			// Requests are denied by default when no rules are defined in the policy (rules == nil) .
			// https://istio.io/latest/docs/reference/config/security/authorization-policy/#AuthorizationPolicy
			Rules:  nil,
			Action: action,
		},
	}
}
