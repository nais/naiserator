package resourcecreator

import (
	"fmt"
	"net/url"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func validateUrl(u *url.URL) error {
	if len(u.Host) == 0 {
		return fmt.Errorf("URL '%s' is missing a hostname", u)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("URL '%s' does not start with 'https://'", u)
	}

	return nil
}

func ingressRule(app *nais.Application, u *url.URL) extensionsv1beta1.IngressRule {
	return extensionsv1beta1.IngressRule{
		Host: u.Host,
		IngressRuleValue: extensionsv1beta1.IngressRuleValue{
			HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
				Paths: []extensionsv1beta1.HTTPIngressPath{
					{
						Path: u.Path,
						Backend: extensionsv1beta1.IngressBackend{
							ServiceName: app.Name,
							ServicePort: intstr.IntOrString{IntVal: nais.DefaultServicePort},
						},
					},
				},
			},
		},
	}
}

func ingressRules(app *nais.Application, urls []string) ([]extensionsv1beta1.IngressRule, error) {
	var rules []extensionsv1beta1.IngressRule

	for _, ingress := range urls {
		parsedUrl, err := url.Parse(ingress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
		}
		if len(parsedUrl.Path) == 0 {
			parsedUrl.Path = "/"
		}
		err = validateUrl(parsedUrl)
		if err != nil {
			return nil, err
		}

		rules = append(rules, ingressRule(app, parsedUrl))
	}

	return rules, nil
}

func Ingress(app *nais.Application) (*extensionsv1beta1.Ingress, error) {

	if len(app.Spec.Ingresses) == 0 {
		return nil, nil
	}

	rules, err := ingressRules(app, app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}

	objectMeta := app.CreateObjectMeta()
	objectMeta.Annotations["prometheus.io/scrape"] = "true"
	objectMeta.Annotations["prometheus.io/path"] = app.Spec.Liveness.Path

	return &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: extensionsv1beta1.IngressSpec{
			Rules: rules,
		},
	}, nil
}
