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

func ingressRules(source Source) ([]networkingv1.IngressRule, error) {
	var rules []networkingv1.IngressRule
	ingresses := source.GetIngress()

	for _, ingress := range ingresses {
		parsedUrl, err := parseIngress(string(ingress))
		if err != nil {
			return nil, err
		}

		rules = append(rules, ingressRule(source.GetName(), parsedUrl))
	}

	return rules, nil
}

func parseIngress(ingress string) (*url.URL, error) {
	parsedUrl, err := url.Parse(ingress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
	}

	if len(parsedUrl.Path) > 1 {
		parsedUrl.Path = strings.TrimRight(parsedUrl.Path, "/") + regexSuffix
	} else {
		parsedUrl.Path = "/"
	}

	err = util.ValidateUrl(parsedUrl)
	if err != nil {
		return nil, err
	}
	return parsedUrl, nil
}

func copyNginxAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "nginx.ingress.kubernetes.io/") {
			dst[k] = v
		}
	}
}

func createIngressBase(source Source) *networkingv1.Ingress {
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
			Rules: []networkingv1.IngressRule{},
		},
	}
}

func createIngressBaseNginx(source Source, ingressClass string, redirect string) (*networkingv1.Ingress, error) {
	var err error
	ingress := createIngressBase(source)
	baseName := fmt.Sprintf("%s-%s", source.GetName(), ingressClass)
	ingress.Name, err = namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	copyNginxAnnotations(ingress.Annotations, source.GetAnnotations())
	ingress.Spec.IngressClassName = &ingressClass

	ingress.Annotations["nginx.ingress.kubernetes.io/use-regex"] = "true"
	ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = backendProtocol(source.GetService().Protocol)

	if redirect != "" {
		ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = redirect + "/$1"
		ingress.Name, err = namegen.ShortName(baseName+"-redirect", validation.DNS1035LabelMaxLength)
		if err != nil {
			return nil, err
		}
	}

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
	rules, err := ingressRules(source)
	if err != nil {
		return nil, err
	}

	ingresses, err := getIngresses(source, cfg, rules, "")
	if err != nil {
		return nil, err
	}

	redirects := source.GetRedirects()
	redirectIngresses := make(map[string]*networkingv1.Ingress)
	if redirects != nil && len(redirects) > 0 {
		err = createRedirectIngresses(source, cfg, redirects, ingresses, redirectIngresses)
		if err != nil {
			return nil, err
		}
	}

	ingressList := make([]*networkingv1.Ingress, 0, len(ingresses)+len(redirectIngresses))
	for _, ingress := range ingresses {
		ingressList = append(ingressList, ingress)
	}

	for _, ingress := range redirectIngresses {
		ingressList = append(ingressList, ingress)
	}

	return ingressList, nil
}

func createRedirectIngresses(source Source, cfg Config, redirects []nais_io_v1.Redirect, ingresses map[string]*networkingv1.Ingress, redirectIngresses map[string]*networkingv1.Ingress) error {
	for _, ing := range ingresses {
		for _, redirect := range redirects {
			for _, rule := range ing.Spec.Rules {
				parsedFromRedirectUrl, err := parseIngress(string(redirect.From))
				if err != nil {
					return err
				}
				parsedToRedirectUrl, err := parseIngress(string(redirect.To))
				if err != nil {
					return err
				}

				// found the ingress that matches the redirect
				if rule.Host == parsedToRedirectUrl.Host {
					u, err := url.Parse(strings.TrimRight(parsedFromRedirectUrl.String(), "/"))
					if err != nil {
						return err
					}

					implementationSpecific := networkingv1.PathTypeImplementationSpecific
					// -V This is an inlined ingressRule call, for readability or something
					r := networkingv1.IngressRule{
						Host: u.Host,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/(.*)?",
										PathType: &implementationSpecific,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: source.GetName(),
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

					ingressClass := util.ResolveIngressClass(parsedFromRedirectUrl.Host, cfg.GetGatewayMappings())
					rdIngress, err := getIngress(source, cfg, r, ingressClass, string(redirect.To))
					if err != nil {
						return err
					}
					redirectIngresses[*ingressClass] = rdIngress
					rdIngress.Spec.Rules = append(rdIngress.Spec.Rules, r)
				}
			}
		}
	}

	if len(redirectIngresses) == 0 {
		return fmt.Errorf("no matching ingress found for redirect")
	}

	return nil
}

func getIngresses(source Source, cfg Config, rules []networkingv1.IngressRule, redirect string) (map[string]*networkingv1.Ingress, error) {
	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil, nil
	}

	ingresses := make(map[string]*networkingv1.Ingress)

	for _, rule := range rules {
		ingressClass := util.ResolveIngressClass(rule.Host, cfg.GetGatewayMappings())
		if ingressClass == nil {
			return nil, fmt.Errorf("the domain %q cannot be used in cluster %q; use one of %v",
				rule.Host,
				cfg.GetClusterName(),
				strings.Join(supportedDomains(cfg.GetGatewayMappings()), ", "),
			)
		}
		ingress := ingresses[*ingressClass]
		if ingress == nil {
			ing, err := getIngress(source, cfg, rule, ingressClass, redirect)
			ingress = ing
			if err != nil {
				return nil, err
			}
			ingresses[*ingressClass] = ingress
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)

	}
	return ingresses, nil
}

func getIngress(source Source, cfg Config, rule networkingv1.IngressRule, ingressClass *string, redirect string) (*networkingv1.Ingress, error) {
	// FIXME: urls in error messages is a nice idea, but needs more planning to avoid tech debt.
	// Reference: __doc_url__/workloads/reference/environments/#ingress-domains
	if ingressClass == nil {
		return nil, fmt.Errorf("the domain %q cannot be used in cluster %q; use one of %v",
			rule.Host,
			cfg.GetClusterName(),
			strings.Join(supportedDomains(cfg.GetGatewayMappings()), ", "),
		)
	}

	ingress, err := createIngressBaseNginx(source, *ingressClass, redirect)
	if err != nil {
		return nil, err
	}
	return ingress, nil
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
