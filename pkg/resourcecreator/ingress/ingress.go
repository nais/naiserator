package ingress

import (
	"fmt"
	"net/url"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
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
	GetIngressClasses(string) ([]string, error)
	GetDomains() []string
	GetDocUrl() string
	GetClusterName() string
}

func createIngressRule(appName string, u *url.URL, isHAProxy bool) networkingv1.IngressRule {
	pathType := networkingv1.PathTypeImplementationSpecific
	if isHAProxy {
		pathType = networkingv1.PathTypePrefix
	}

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

func createIngressBase(source Source, ingressClass string) (*networkingv1.Ingress, error) {
	baseName := fmt.Sprintf("%s-%s", source.GetName(), ingressClass)
	shortName, err := namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}

	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = shortName
	objectMeta.Annotations["prometheus.io/scrape"] = "true"
	objectMeta.Annotations["prometheus.io/path"] = source.GetLiveness().Path

	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: objectMeta,
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules:            []networkingv1.IngressRule{},
		},
	}, nil
}

func createIngressBaseHAProxy(source Source, ingressClass string) (*networkingv1.Ingress, error) {
	return createIngressBase(source, ingressClass)
}

func createIngressBaseNginx(source Source, ingressClass string) (*networkingv1.Ingress, error) {
	ingress, err := createIngressBase(source, ingressClass)
	if err != nil {
		return nil, err
	}

	copyNginxAnnotations(ingress.Annotations, source.GetAnnotations())
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

func createIngressList(source Source, cfg Config) ([]*networkingv1.Ingress, error) {
	ingresses, err := createIngresses(source, cfg)
	if err != nil {
		return nil, err
	}

	redirectIngresses := make(map[string]*networkingv1.Ingress)
	if hasRedirects(source) {
		err := createRedirectIngresses(source, cfg, ingresses, redirectIngresses)
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

func createIngress(source Source, ingressClass string, isHAProxy bool) (*networkingv1.Ingress, error) {
	if isHAProxy {
		return createIngressBaseHAProxy(source, ingressClass)
	} else {
		return createIngressBaseNginx(source, ingressClass)
	}
}

func createIngresses(source Source, cfg Config) (map[string]*networkingv1.Ingress, error) {
	ingresses := make(map[string]*networkingv1.Ingress)

	for _, ingress := range source.GetIngress() {
		parsedUrl, err := parseIngress(string(ingress))
		if err != nil {
			return nil, err
		}

		ingressClasses, err := cfg.GetIngressClasses(parsedUrl.Host)
		if err != nil {
			return nil, err
		}

		for _, ingressClass := range ingressClasses {
			isHAProxy := strings.HasSuffix(ingressClass, "haproxy")

			ingress := ingresses[ingressClass]
			if ingress == nil {
				newIngress, err := createIngress(source, ingressClass, isHAProxy)
				if err != nil {
					return nil, err
				}

				ingress = newIngress
				ingresses[ingressClass] = ingress
			}

			rule := createIngressRule(source.GetName(), parsedUrl, isHAProxy)
			ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
		}
	}

	return ingresses, nil
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	if len(source.GetIngress()) == 0 {
		return nil
	}

	ingresses, err := createIngressList(source, cfg)
	if err != nil {
		return err
	}

	for _, ing := range ingresses {
		ast.AppendOperation(resource.OperationCreateOrUpdate, ing)
	}
	return nil
}
