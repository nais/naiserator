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

func copyHAProxyAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if strings.HasPrefix(k, "haproxy.org/") {
			dst[k] = v
		}
	}
}

func migrateNginxAnnotationsToHAProxyAnnotations(haProxy, nginx map[string]string) {
	nginxAnnotations := map[string]string{}
	copyNginxAnnotations(nginxAnnotations, nginx)

	// Mapping from nginx annotation short key to haproxy annotation short key.
	// Empty string means no direct equivalent exists yet.
	haProxyAnnotations := map[string]string{
		"proxy-read-timeout":    "timeout-server",
		"proxy-send-timeout":    "timeout-client",
		"proxy-connect-timeout": "timeout-connect",
		"rewrite-target":        "path-rewrite", // TODO: Her må det kodes litt

		// "permanent-redirect":    "",             // TODO: no direct equivalent, dette fikser name: sfs-legacy-redirect-ingress, namespace: teamsykmelding
		// "whitelist-source-range":     "allow-list", // TODO: brukt av atil

		// "proxy-body-size":       "",             // no direct equivalent; use backend-config-snippet with http-request deny
		// "use-regex":                  "",                       // no direct equivalent; HAProxy uses path-rewrite or path-regex
		// "backend-protocol":           "server-ssl",             // value differs: nginx "HTTPS" → haproxy "enabled"
		// "secure-backends":            "server-ssl",             // value differs: nginx "true" → haproxy "enabled"
		// "proxy-ssl-verify":           "server-ssl-verify",      // value differs: nginx "on"/"off" → haproxy "enabled"/"disabled"
		// "proxy-next-upstream-timeout": "timeout-server",        // closest equivalent; no direct 1:1 mapping
		// "proxy-next-upstream-tries":  "",                       // TODO: use backend-config-snippet with "retries 3"
		// "server-snippet":             "",                       // no direct equivalent; use backend-config-snippet manually
		// "configuration-snippet":      "frontend-config-snippet",
		// "upstream-vhost":             "set-host",
		// "from-to-www-redirect":       "",                       // no direct equivalent
		// "enable-global-auth":         "",                       // no direct equivalent; use auth-type + auth-secret
		// "proxy-buffer-size":          "",                       // no direct equivalent; tune.bufsize via global config
		// "proxy-buffers-number":       "",                       // no direct equivalent; tune.bufsize via global config
		// "proxy-busy-buffers-size":    "",                       // no direct equivalent
		// "large-client-header-buffers": "",                      // no direct equivalent; tune.bufsize via global config
		// "limit-rpm":                  "",                       // TODO: use rate-limit-requests + rate-limit-period
		// "limit-rps":                  "",                       // TODO: use rate-limit-requests + rate-limit-period
		// "limit-burst-multiplier":     "",                       // TODO: use rate-limit-requests + rate-limit-period
		// "limit-connections":          "",                       // TODO: stick-table based rate limiting
		// "denylist-source-range":      "deny-list",
	}

	for key, value := range nginxAnnotations {
		nginxKey, _ := strings.CutPrefix(key, "nginx.ingress.kubernetes.io/")
		haProxyKey, ok := haProxyAnnotations[nginxKey]
		if ok && haProxyKey != "" {
			haProxy["haproxy.org/"+haProxyKey] = value
		}
	}
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
	ingress, err := createIngressBase(source, ingressClass)
	if err != nil {
		return nil, err
	}

	migrateNginxAnnotationsToHAProxyAnnotations(ingress.Annotations, source.GetAnnotations())
	copyHAProxyAnnotations(ingress.Annotations, source.GetAnnotations())

	return ingress, nil
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
