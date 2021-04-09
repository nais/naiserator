package resourcecreator

import (
	"fmt"
	"net/url"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/util"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

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

func NginxIngresses(app *nais.Application, options ResourceOptions) ([]*networkingv1beta1.Ingress, error) {
	var i int

	rules, err := ingressRules(app, app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}

	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil, nil
	}

	createIngressBase := func(host string) (*networkingv1beta1.Ingress, error) {
		i++
		ingress := ingressBase(app)
		ingress.Name = fmt.Sprintf("%s-%02d", app.Name, i)
		ingressClass := ResolveIngressClass(host, options.GatewayMappings)
		if ingressClass == nil {
			return nil, fmt.Errorf("domain '%s' is not supported", host)
		}
		ingress.Annotations["kubernetes.io/ingress.class"] = *ingressClass
		return ingress, nil
	}

	ingresses := make(map[string]*networkingv1beta1.Ingress)

	for _, rule := range rules {
		ingress := ingresses[rule.Host]
		if ingress == nil {
			ingress, err = createIngressBase(rule.Host)
			if err != nil {
				return nil, err
			}
			ingresses[rule.Host] = ingress
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
	}

	ingressList := make([]*networkingv1beta1.Ingress, 0, len(ingresses))
	for _, ingress := range ingresses {
		ingressList = append(ingressList, ingress)
	}
	return ingressList, nil
}
