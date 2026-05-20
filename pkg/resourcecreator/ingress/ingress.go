package ingress

import (
	"fmt"
	"net/url"
	"regexp"
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
	regexSuffix                    = "(/.*)?"
	haproxyBackendConfigAnnotation = "haproxy.org/backend-config-snippet"
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
	IsHAProxyEnabled() bool
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
	parsedURL, err := url.Parse(ingress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
	}

	if len(parsedURL.Path) > 1 {
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	} else {
		parsedURL.Path = "/"
	}

	err = util.ValidateUrl(parsedURL)
	if err != nil {
		return nil, err
	}
	return parsedURL, nil
}

func copyHAProxyAnnotations(dst, src map[string]string) {
	for k, v := range src {
		if !strings.HasPrefix(k, "haproxy.org/") {
			continue
		}

		if k == haproxyBackendConfigAnnotation {
			if existing := dst[k]; existing != "" {
				dst[k] = fmt.Sprintf("%s\n\n%s", v, existing)
				continue
			}
		}
		dst[k] = v
	}
}

func migrateNginxAnnotationsToHAProxyAnnotations(haProxy, nginx map[string]string) {
	nginxAnnotations := map[string]string{}
	copyNginxAnnotations(nginxAnnotations, nginx)

	// Mapping from nginx annotation short key to HAProxy equivalent.
	// Only timeout annotations are migrated; other nginx annotations
	// (e.g. permanent-redirect, proxy-body-size)
	// have no direct HAProxy equivalent and are not propagated by naiserator.
	haProxyAnnotations := map[string]string{
		"keepalive-timeout":     "timeout-http-keep-alive",
		"proxy-connect-timeout": "timeout-connect",
		"proxy-read-timeout":    "timeout-server",
		"proxy-send-timeout":    "timeout-client",
		"upstream-vhost":        "set-host",
	}

	for key, value := range nginxAnnotations {
		nginxKey, _ := strings.CutPrefix(key, "nginx.ingress.kubernetes.io/")
		haProxyKey, performMapping := haProxyAnnotations[nginxKey]
		if !performMapping || haProxyKey == "" {
			continue
		}

		haProxy["haproxy.org/"+haProxyKey] = value
		if strings.HasSuffix(nginxKey, "-timeout") && strings.HasPrefix(haProxyKey, "timeout-") {
			haProxy["haproxy.org/"+haProxyKey] = value + "s"
		}
	}

	migrateLimitRpm(haProxy, nginxAnnotations)
	migrateRewriteTarget(haProxy, nginxAnnotations)
	migrateWhitelistSourceRange(haProxy, nginxAnnotations)
}

var (
	nginxCaptureGroupRegex = regexp.MustCompile(`\$(\d+)`)
	nginxArgVariableRegex  = regexp.MustCompile(`\$arg_(\w+)`)
)

func migrateLimitRpm(annotations, nginxAnnotations map[string]string) {
	limitRpm := nginxAnnotations["nginx.ingress.kubernetes.io/limit-rpm"]
	if limitRpm == "" {
		return
	}

	annotations["haproxy.org/rate-limit-period"] = "1m"
	annotations["haproxy.org/rate-limit-request"] = fmt.Sprint(limitRpm)
	annotations["haproxy.org/rate-limit-status-code"] = "429"
}

// migrateRewriteTarget translates an nginx rewrite-target annotation to equivalent HAProxy annotation(s)
func migrateRewriteTarget(annotations, nginxAnnotations map[string]string) {
	rewriteTarget := nginxAnnotations["nginx.ingress.kubernetes.io/rewrite-target"]
	if rewriteTarget == "" {
		return
	}

	isAbsolute := strings.HasPrefix(rewriteTarget, "http://") || strings.HasPrefix(rewriteTarget, "https://")

	if !isAbsolute {
		annotations["haproxy.org/path-rewrite"] = nginxToHAProxyCaptureGroups(rewriteTarget)
		return
	}

	// Absolute URL → nginx sends 302.
	// HAProxy's request-redirect only supports host substitution and preserves original URI,
	// so path-changing redirects must use backend-config-snippet.
	u, err := url.Parse(rewriteTarget)
	if err != nil {
		migrateRewriteTargetAsSnippet(annotations, rewriteTarget)
		return
	}

	hostOnly := u.Path == "" || u.Path == "/"
	hasCaptures := nginxCaptureGroupRegex.MatchString(rewriteTarget)
	hasArgs := nginxArgVariableRegex.MatchString(rewriteTarget)

	if hostOnly && !hasCaptures && !hasArgs {
		annotations["haproxy.org/request-redirect"] = u.Host
		annotations["haproxy.org/request-redirect-code"] = "302"
		return
	}

	migrateRewriteTargetAsSnippet(annotations, rewriteTarget)
}

func migrateWhitelistSourceRange(annotations, nginxAnnotations map[string]string) {
	whitelistSourceRange := nginxAnnotations["nginx.ingress.kubernetes.io/whitelist-source-range"]
	if whitelistSourceRange == "" {
		return
	}

	ranges := []string{}
	for sourceRange := range strings.SplitSeq(whitelistSourceRange, ",") {
		sourceRange = strings.TrimSpace(sourceRange)
		if sourceRange != "" {
			ranges = append(ranges, sourceRange)
		}
	}
	if len(ranges) == 0 {
		return
	}

	snippet := fmt.Sprintf(
		"acl allowed_src req.hdr_ip(X-Forwarded-For) %s\nhttp-request deny deny_status 403 if !allowed_src",
		strings.Join(ranges, " "),
	)
	delimiter := "# Added by naiserator due to the `nginx.ingress.kubernetes.io/whitelist-source-range` annotation:"
	snippet = fmt.Sprintf("###\n%s\n%s\n###", delimiter, snippet)

	if existing := annotations[haproxyBackendConfigAnnotation]; existing != "" {
		annotations[haproxyBackendConfigAnnotation] = fmt.Sprintf("%s\n\n%s", existing, snippet)
		return
	}
	annotations[haproxyBackendConfigAnnotation] = snippet
}

// nginxToHAProxyCaptureGroups converts $1, $2 to \1, \2
func nginxToHAProxyCaptureGroups(s string) string {
	return nginxCaptureGroupRegex.ReplaceAllString(s, `\$1`)
}

// migrateRewriteTargetAsSnippet uses backend-config-snippet for complex redirects
func migrateRewriteTargetAsSnippet(annotations map[string]string, rewriteTarget string) {
	haproxyTarget := nginxToHAProxyCaptureGroups(rewriteTarget)
	haproxyTarget = nginxArgVariableRegex.ReplaceAllString(haproxyTarget, `%[url_param($1)]`)
	snippet := fmt.Sprintf("http-request redirect location %s code 302", haproxyTarget)
	delimiter := "# Added by naiserator due to the `nginx.ingress.kubernetes.io/rewrite-target` annotation:"
	snippet = fmt.Sprintf("###\n%s\n%s\n###", delimiter, snippet)

	if existing := annotations[haproxyBackendConfigAnnotation]; existing != "" {
		annotations[haproxyBackendConfigAnnotation] = fmt.Sprintf("%s\n\n%s", existing, snippet)
		return
	}
	annotations[haproxyBackendConfigAnnotation] = snippet
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
		parsedURL, err := parseIngress(string(ingress))
		if err != nil {
			return nil, err
		}

		ingressClasses, err := cfg.GetIngressClasses(parsedURL.Host)
		if err != nil {
			return nil, err
		}

		for _, ingressClass := range ingressClasses {
			isHAProxy := strings.HasSuffix(ingressClass, "haproxy")

			if isHAProxy && !cfg.IsHAProxyEnabled() {
				continue // only create HAProxy ingress where HAProxy is enabled
			}

			ingress := ingresses[ingressClass]
			if ingress == nil {
				newIngress, err := createIngress(source, ingressClass, isHAProxy)
				if err != nil {
					return nil, err
				}

				ingress = newIngress
				ingresses[ingressClass] = ingress
			}

			ruleURL := parsedURL
			if !isHAProxy && len(parsedURL.Path) > 1 { // handle Nginx - delete block on Nginx sunsetting
				nginxURL := *parsedURL
				nginxURL.Path = parsedURL.Path + regexSuffix
				ruleURL = &nginxURL
			}

			rule := createIngressRule(source.GetName(), ruleURL, isHAProxy)
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
