package ingress

import (
	"fmt"
	"net/url"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	regexSuffix = "(/.*)?"
)

type Source interface {
	resource.Source
	GetIngress() []nais_io_v1.Ingress
	GetRedirects() []nais_io_v1.Redirect
	GetLiveness() *nais_io_v1.Probe
	GetService() *nais_io_v1.Service
}

type Config interface {
	GetGatewayMappings() []config.GatewayMapping
	IsLinkerdEnabled() bool
	GetDocUrl() string
	GetClusterName() string
}

func ingressRule(appName string, u *url.URL) networkingv1.IngressRule {
	pathType := networkingv1.PathTypeImplementationSpecific

	return networkingv1.IngressRule{
		Host: u.Host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     u.Path,
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: appName,
								Port: networkingv1.ServiceBackendPort{
									Number: int32(nais_io_v1alpha1.DefaultServicePort),
								},
							},
						},
					},
				},
			},
		},
	}
}

func ingressRules(source Source) ([]networkingv1.IngressRule, []networkingv1.IngressRule, error) {
	var rules []networkingv1.IngressRule
	var redirectRules []networkingv1.IngressRule
	redirects := source.GetRedirects()
	ingresses := source.GetIngress()

	for _, redirect := range redirects {
		parsedUrl, err := url.Parse(string(redirect.To))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse URL '%s': %s", redirect.To, err)
		}

		if len(parsedUrl.Path) > 1 {
			parsedUrl.Path = strings.TrimRight(parsedUrl.Path, "/") + regexSuffix
		} else {
			parsedUrl.Path = "/"
		}

		err = util.ValidateUrl(parsedUrl)
		if err != nil {
			return nil, nil, err
		}

		redirectRules = append(redirectRules, ingressRule(source.GetName(), parsedUrl))
	}

	for _, ingress := range ingresses {
		parsedUrl, err := url.Parse(string(ingress))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
		}

		if len(parsedUrl.Path) > 1 {
			parsedUrl.Path = strings.TrimRight(parsedUrl.Path, "/") + regexSuffix
		} else {
			parsedUrl.Path = "/"
		}

		err = util.ValidateUrl(parsedUrl)
		if err != nil {
			return nil, nil, err
		}

		rules = append(rules, ingressRule(source.GetName(), parsedUrl))
	}

	return rules, redirectRules, nil
}

func copyNginxAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "nginx.ingress.kubernetes.io/") {
			dst[k] = v
		}
	}
}

func createIngressBase(source Source, rules []networkingv1.IngressRule) *networkingv1.Ingress {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Annotations["prometheus.io/scrape"] = "true"
	objectMeta.Annotations["prometheus.io/path"] = source.GetLiveness().Path

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: networkingv1.IngressSpec{
			Rules: rules,
		},
	}
}

func createIngressBaseNginx(source Source, ingressClass string) (*networkingv1.Ingress, error) {
	var err error
	ingress := createIngressBase(source, []networkingv1.IngressRule{})
	baseName := fmt.Sprintf("%s-%s", source.GetName(), ingressClass)
	ingress.Name, err = namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	copyNginxAnnotations(ingress.Annotations, source.GetAnnotations())
	ingress.Spec.IngressClassName = &ingressClass

	ingress.Annotations["nginx.ingress.kubernetes.io/use-regex"] = "true"
	ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = backendProtocol(source.GetService().Protocol)

	return ingress, nil
}

// Using backend-protocol annotations is possible to indicate how NGINX should communicate with the backend service.
// Valid Values: HTTP, HTTPS, GRPC, GRPCS, AJP and FCGI
// By default NGINX uses HTTP.
// URL: https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-protocol
func backendProtocol(portName string) string {
	switch portName {
	case "grpc":
		return "GRPC"
	default:
		return "HTTP"
	}
}

func supportedDomains(gatewayMappings []config.GatewayMapping) []string {
	domains := make([]string, 0, len(gatewayMappings))

	for _, v := range gatewayMappings {
		domains = append(domains, v.DomainSuffix)
	}
	return domains
}

func nginxIngresses(source Source, cfg Config) ([]*networkingv1.Ingress, error) {
	rules, redirectRules, err := ingressRules(source)
	if err != nil {
		return nil, err
	}

	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil, nil
	}

	ingresses := make(map[string]*networkingv1.Ingress)

	for _, rule := range rules {
		ingressClass := util.ResolveIngressClass(rule.Host, cfg.GetGatewayMappings())

		// FIXME: urls in error messages is a nice idea, but needs more planning to avoid tech debt.
		// Reference: __doc_url__/workloads/reference/environments/#ingress-domains
		if ingressClass == nil {
			return nil,
				fmt.Errorf("the domain %q cannot be used in cluster %q; use one of %v",
					rule.Host,
					cfg.GetClusterName(),
					strings.Join(supportedDomains(cfg.GetGatewayMappings()), ", "),
				)
		}

		ingress := ingresses[*ingressClass]
		if ingress == nil {
			ingress, err = createIngressBaseNginx(source, *ingressClass)
			if err != nil {
				return nil, err
			}
			ingresses[*ingressClass] = ingress
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
	}

	ingressList := make([]*networkingv1.Ingress, 0, len(ingresses))
	for _, ingress := range ingresses {
		ingressList = append(ingressList, ingress)
	}

	// redirects := source.GetRedirects()

	// for _, redirect := range redirects {

	// 	ingressClass := util.ResolveIngressClass(rule.Host, cfg.GetGatewayMappings())
	// }

	return ingressList, nil
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	ingresses, err := nginxIngresses(source, cfg)
	if err != nil {
		return err
	}

	for _, ing := range ingresses {
		ast.AppendOperation(resource.OperationCreateOrUpdate, ing)
	}
	return nil
}
