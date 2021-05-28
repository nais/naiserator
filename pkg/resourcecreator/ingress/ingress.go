package ingress

import (
	"fmt"
	"net/url"
	"strings"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
)

const regexSuffix = "(/.*)?"

func ingressRule(appName string, u *url.URL) networkingv1beta1.IngressRule {
	return networkingv1beta1.IngressRule{
		Host: u.Host,
		IngressRuleValue: networkingv1beta1.IngressRuleValue{
			HTTP: &networkingv1beta1.HTTPIngressRuleValue{
				Paths: []networkingv1beta1.HTTPIngressPath{
					{
						Path: u.Path,
						Backend: networkingv1beta1.IngressBackend{
							ServiceName: appName,
							ServicePort: intstr.IntOrString{IntVal: nais_io_v1alpha1.DefaultServicePort},
						},
					},
				},
			},
		},
	}
}

func ingressRules(objectMeta metav1.ObjectMeta, naisIngresses []nais_io_v1alpha1.Ingress) ([]networkingv1beta1.IngressRule, error) {
	var rules []networkingv1beta1.IngressRule

	for _, ingress := range naisIngresses {
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

		rules = append(rules, ingressRule(objectMeta.Name, parsedUrl))
	}

	return rules, nil
}

func ingressRulesNginx(objectMeta metav1.ObjectMeta, naisIngresses []nais_io_v1alpha1.Ingress) ([]networkingv1beta1.IngressRule, error) {
	var rules []networkingv1beta1.IngressRule

	for _, ingress := range naisIngresses {
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

		rules = append(rules, ingressRule(objectMeta.Name, parsedUrl))
	}

	return rules, nil
}

func copyNginxAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "nginx.ingress.kubernetes.io/") {
			dst[k] = v
		}
	}
}

func createIngressBase(objectMeta metav1.ObjectMeta, rules []networkingv1beta1.IngressRule, livenessPath string) *networkingv1beta1.Ingress {
	objectMeta.Annotations["prometheus.io/scrape"] = "true"
	objectMeta.Annotations["prometheus.io/path"] = livenessPath

	return &networkingv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1beta1",
		},
		ObjectMeta: *objectMeta.DeepCopy(),
		Spec: networkingv1beta1.IngressSpec{
			Rules: rules,
		},
	}
}

func createIngressBaseNginx(objectMeta metav1.ObjectMeta, ingressClass, livenessPath, serviceProtocol string, naisAnnotations map[string]string) (*networkingv1beta1.Ingress, error) {
	var err error
	ingress := createIngressBase(objectMeta, []networkingv1beta1.IngressRule{}, livenessPath)
	baseName := fmt.Sprintf("%s-%s", objectMeta.Name, ingressClass)
	ingress.Name, err = namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	copyNginxAnnotations(ingress.Annotations, naisAnnotations)

	ingress.Annotations["kubernetes.io/ingress.class"] = ingressClass
	ingress.Annotations["nginx.ingress.kubernetes.io/use-regex"] = "true"
	ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = backendProtocol(serviceProtocol)
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

func nginxIngresses(objectMeta metav1.ObjectMeta, options resource.Options, naisIngresses []nais_io_v1alpha1.Ingress, livenessPath, serviceProtocol string, naisAnnotations map[string]string) ([]*networkingv1beta1.Ingress, error) {
	rules, err := ingressRulesNginx(objectMeta, naisIngresses)
	if err != nil {
		return nil, err
	}

	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil, nil
	}

	ingresses := make(map[string]*networkingv1beta1.Ingress)

	for _, rule := range rules {
		ingressClass := util.ResolveIngressClass(rule.Host, options.GatewayMappings)
		if ingressClass == nil {
			return nil, fmt.Errorf("domain '%s' is not supported", rule.Host)
		}
		ingress := ingresses[*ingressClass]
		if ingress == nil {
			ingress, err = createIngressBaseNginx(objectMeta, *ingressClass, livenessPath, serviceProtocol, naisAnnotations)
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

func linkerdIngresses(objectMeta metav1.ObjectMeta, options resource.Options, operations *resource.Operations, naisIngresses []nais_io_v1alpha1.Ingress, livenessPath, serviceProtocol string, naisAnnotations map[string]string) error {
	ingresses, err := nginxIngresses(objectMeta, options, naisIngresses, livenessPath, serviceProtocol, naisAnnotations)
	if err != nil {
		return fmt.Errorf("while creating ingresses: %s", err)
	}

	if ingresses != nil {
		for _, ing := range ingresses {
			*operations = append(*operations, resource.Operation{Resource: ing, Operation: resource.OperationCreateOrUpdate})
		}
	}
	return nil
}

func onPremIngresses(objectMeta metav1.ObjectMeta, operations *resource.Operations, naisIngresses []nais_io_v1alpha1.Ingress, livenessPath string) error {
	rules, err := ingressRules(objectMeta, naisIngresses)
	if err != nil {
		return err
	}

	// Ingress objects must have at least one path rule to be valid.
	if len(rules) == 0 {
		return nil
	}

	ingress := createIngressBase(objectMeta, rules, livenessPath)
	*operations = append(*operations, resource.Operation{Resource: ingress, Operation: resource.OperationCreateOrUpdate})
	return nil
}

func Create(objectMeta metav1.ObjectMeta, options resource.Options, operations *resource.Operations, naisIngresses []nais_io_v1alpha1.Ingress, livenessPath, serviceProtocol string, naisAnnotations map[string]string) error {
	if options.Linkerd {
		err := linkerdIngresses(objectMeta, options, operations, naisIngresses, livenessPath, serviceProtocol, naisAnnotations)
		if err != nil {
			return err
		}
	} else {
		err := onPremIngresses(objectMeta, operations, naisIngresses, livenessPath)
		if err != nil {
			return err
		}
	}

	return nil
}
