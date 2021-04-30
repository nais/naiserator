package ingress

import (
	"fmt"
	"net/url"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resourceutils"
	"github.com/nais/naiserator/pkg/util"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

const regexSuffix = "(/.*)?"

func ingressRule(app *nais.Application, u *url.URL) networkingv1beta1.IngressRule {
	return networkingv1beta1.IngressRule{
		Host: u.Host,
		IngressRuleValue: networkingv1beta1.IngressRuleValue{
			HTTP: &networkingv1beta1.HTTPIngressRuleValue{
				Paths: []networkingv1beta1.HTTPIngressPath{
					{
						Path: u.Path,
						Backend: networkingv1beta1.IngressBackend{
							ServiceName: app.Name,
							ServicePort: intstr.IntOrString{IntVal: nais.DefaultServicePort},
						},
					},
				},
			},
		},
	}
}

func ingressRules(app *nais.Application, urls []nais.Ingress) ([]networkingv1beta1.IngressRule, error) {
	var rules []networkingv1beta1.IngressRule

	for _, ingress := range urls {
		parsedUrl, err := url.Parse(string(ingress))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
		}
		if len(parsedUrl.Path) == 0 {
			parsedUrl.Path = "/"
		}
		err = util.ValidateUrl(parsedUrl)
		if err != nil {
			return nil, err
		}

		rules = append(rules, ingressRule(app, parsedUrl))
	}

	return rules, nil
}

func ingressRulesNginx(app *nais.Application) ([]networkingv1beta1.IngressRule, error) {
	var rules []networkingv1beta1.IngressRule

	for _, ingress := range app.Spec.Ingresses {
		parsedUrl, err := url.Parse(string(ingress))
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
		}

		if len(parsedUrl.Path) > 1 {
			err = util.ValidateUrl(parsedUrl)
			if err != nil {
				return nil, err
			}
			parsedUrl.Path = strings.TrimRight(parsedUrl.Path, "/") + regexSuffix
		} else {
			parsedUrl.Path = "/"
		}

		rules = append(rules, ingressRule(app, parsedUrl))
	}

	return rules, nil
}

func ingressBase(app *nais.Application) *networkingv1beta1.Ingress {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Annotations["prometheus.io/scrape"] = "true"
	objectMeta.Annotations["prometheus.io/path"] = app.Spec.Liveness.Path

	return &networkingv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{},
		},
	}
}

func Ingress(app *nais.Application) (*networkingv1beta1.Ingress, error) {
	rules, err := ingressRules(app, app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}

	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil, nil
	}

	objectMeta := app.CreateObjectMeta()
	objectMeta.Annotations["prometheus.io/scrape"] = "true"
	objectMeta.Annotations["prometheus.io/path"] = app.Spec.Liveness.Path

	return &networkingv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: networkingv1beta1.IngressSpec{
			Rules: rules,
		},
	}, nil
}

func ResolveIngressClass(host string, mappings []config.GatewayMapping) *string {
	for _, mapping := range mappings {
		if strings.HasSuffix(host, mapping.DomainSuffix) {
			return &mapping.IngressClass
		}
	}
	return nil
}

// https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-protocol
// Using backend-protocol annotations is possible to indicate how NGINX should communicate with the backend service.
// Valid Values: HTTP, HTTPS, GRPC, GRPCS, AJP and FCGI
// By default NGINX uses HTTP.
func backendProtocol(portName string) string {
	switch portName {
	case "grpc":
		return "GRPC"
	default:
		return "HTTP"
	}
}

func NginxIngresses(app *nais.Application, options resourceutils.Options) ([]*networkingv1beta1.Ingress, error) {
	rules, err := ingressRulesNginx(app)
	if err != nil {
		return nil, err
	}

	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil, nil
	}

	createIngressBase := func(host, ingressClass string) (*networkingv1beta1.Ingress, error) {
		ingress := ingressBase(app)
		baseName := fmt.Sprintf("%s-%s", app.Name, ingressClass)
		ingress.Name, err = namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
		if err != nil {
			return nil, err
		}
		ingress.Annotations["kubernetes.io/ingress.class"] = ingressClass
		ingress.Annotations["nginx.ingress.kubernetes.io/use-regex"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = backendProtocol(app.Spec.Service.Protocol)
		return ingress, nil
	}

	ingresses := make(map[string]*networkingv1beta1.Ingress)

	for _, rule := range rules {
		ingressClass := ResolveIngressClass(rule.Host, options.GatewayMappings)
		if ingressClass == nil {
			return nil, fmt.Errorf("domain '%s' is not supported", rule.Host)
		}
		ingress := ingresses[*ingressClass]
		if ingress == nil {
			ingress, err = createIngressBase(rule.Host, *ingressClass)
			if err != nil {
				return nil, err
			}
			ingresses[*ingressClass] = ingress
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
	}

	ingressList := make([]*networkingv1beta1.Ingress, 0, len(ingresses))
	for _, ingress := range ingresses {
		ingressList = append(ingressList, ingress)
	}
	return ingressList, nil
}
