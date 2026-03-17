package ingress

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AllowedRedirectAnnotation = "nais.io/allow-redirect"
)

func hasRedirects(source Source) bool {
	return source.GetRedirects() != nil && len(source.GetRedirects()) > 0
}

func findMatch(ingresses []networkingv1.Ingress, redirectFromHost string) *networkingv1.Ingress {
	for _, ing := range ingresses {
		for _, rule := range ing.Spec.Rules {
			if rule.Host == redirectFromHost {
				return &ing
			}
		}
	}
	return nil
}

func allowedRedirectAnnotation(ctx context.Context, kube client.Client, matchedIngress *networkingv1.Ingress) error {
	appName := matchedIngress.Labels["app"]
	if len(appName) == 0 {
		return fmt.Errorf("no 'app' label found in the ingress in namespace %s", matchedIngress.Namespace)
	}

	// Retrieve app object in the matched ingress namespace
	app := &nais_io_v1alpha1.Application{}
	appKey := client.ObjectKey{Name: appName, Namespace: matchedIngress.Namespace}
	err := kube.Get(ctx, appKey, app)
	if err != nil {
		return fmt.Errorf("failed to get app object in namespace %s: %s", matchedIngress.Namespace, err)
	}

	// Validate annotation
	if app.Annotations[AllowedRedirectAnnotation] != "true" {
		return fmt.Errorf("cross-namespace redirect not allowed from app '%s' without annotation 'nais.io/allow-redirect: true' in namespace '%s'", app.GetName(), matchedIngress.Namespace)
	}
	return nil
}

func RedirectAllowed(ctx context.Context, source Source, kube client.Client) error {
	if !hasRedirects(source) {
		return nil
	}

	for _, redirect := range source.GetRedirects() {
		redirectFrom, err := url.Parse(string(redirect.From))
		if err != nil {
			return fmt.Errorf("failed to parse redirect URL: %s", err)
		}
		ingressList := &networkingv1.IngressList{}

		// Search for ingresses in the application's namespace first
		listOptions := &client.ListOptions{Namespace: source.GetNamespace()}
		err = kube.List(ctx, ingressList, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list ingresses in namespace %s: %s", source.GetNamespace(), err)
		}
		matchedIngress := findMatch(ingressList.Items, redirectFrom.Host)

		if matchedIngress == nil {
			// No match found in the same namespace; search across all namespaces
			listOptions = &client.ListOptions{}
			err = kube.List(ctx, ingressList, listOptions)
			if err != nil {
				return fmt.Errorf("failed to list ingresses: %s", err)
			}

			// Find ingress in other namespaces
			matchedIngress = findMatch(ingressList.Items, redirectFrom.Host)
			if matchedIngress == nil {
				return fmt.Errorf("no ingress found with host matching redirect from URL: %s", redirectFrom)
			}

			// If the ingress is in a different namespace, check for the required annotation
			if matchedIngress.Namespace != source.GetNamespace() {
				if err = allowedRedirectAnnotation(ctx, kube, matchedIngress); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func createRedirectIngressRule(source Source, redirectUrl string, isHAProxy bool) (networkingv1.IngressRule, error) {
	u, err := url.Parse(strings.TrimRight(redirectUrl, "/"))
	if err != nil {
		return networkingv1.IngressRule{}, nil
	}

	path := "/(.*)?"
	pathType := networkingv1.PathTypeImplementationSpecific
	if isHAProxy {
		path = "/"
		pathType = networkingv1.PathTypePrefix
	}

	return networkingv1.IngressRule{
		Host: u.Host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     path,
						PathType: &pathType,
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
	}, nil
}

func addRedirectConfiguration(source Source, ingressClass string, ingress *networkingv1.Ingress, redirect *url.URL, isHAProxy bool) error {
	var err error
	baseName := fmt.Sprintf("%s-%s", source.GetName(), ingressClass)
	ingress.Name, err = namegen.ShortName(baseName+"-redirect", validation.DNS1035LabelMaxLength)
	if err != nil {
		return err
	}

	if isHAProxy {
		ingress.Annotations["haproxy.org/request-redirect"] = redirect.Host
		ingress.Annotations["haproxy.org/request-redirect-code"] = "302"
	} else {
		ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = redirect.String() + "$1"
	}

	return nil
}

func createRedirectIngresses(source Source, cfg Config, ingresses map[string]*networkingv1.Ingress, redirectIngresses map[string]*networkingv1.Ingress) error {
	redirects := source.GetRedirects()
	for _, ing := range ingresses {
		for _, redirect := range redirects {
			parsedToRedirectUrl, err := parseIngress(string(redirect.To))
			if err != nil {
				return err
			}

			for _, ingressRule := range ing.Spec.Rules {
				if ingressRule.Host != parsedToRedirectUrl.Host {
					continue
				}

				parsedFromRedirectUrl, err := parseIngress(string(redirect.From))
				if err != nil {
					return err
				}

				ingressClasses, err := cfg.GetIngressClasses(parsedFromRedirectUrl.Host)
				if err != nil {
					return err
				}

				for _, ingressClass := range ingressClasses {
					isHAProxy := strings.HasSuffix(ingressClass, "haproxy")

					rule, err := createRedirectIngressRule(source, parsedFromRedirectUrl.String(), isHAProxy)
					if err != nil {
						return err
					}

					ingress, err := createIngress(source, ingressClass, isHAProxy)
					if err != nil {
						return err
					}

					if err := addRedirectConfiguration(source, ingressClass, ingress, parsedToRedirectUrl, isHAProxy); err != nil {
						return err
					}

					redirectIngresses[ingressClass] = ingress
					ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
				}
			}
		}
	}

	if len(redirectIngresses) == 0 {
		return fmt.Errorf("no matching ingress found for redirect")
	}

	return nil
}
