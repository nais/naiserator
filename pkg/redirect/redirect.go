package redirect

import (
	"context"
	"fmt"
	"net/url"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AllowedRedirectAnnotation = "nais.io/allow-redirect"
)

func hasRedirect(app *nais_io_v1alpha1.Application) bool {
	return app.GetRedirects() != nil && len(app.GetRedirects()) > 0
}

func checkAppForAllowedRedirectAnnotation(ctx context.Context, kube client.Client, matchedIngress *networkingv1.Ingress) error {
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

func Allowed(ctx context.Context, app *nais_io_v1alpha1.Application, kube client.Client) error {
	if !hasRedirect(app) {
		return nil
	}

	for _, redirect := range app.GetRedirects() {
		redirectFrom, err := url.Parse(string(redirect.From))
		if err != nil {
			return fmt.Errorf("failed to parse redirect URL: %s", err)
		}
		ingressList := &networkingv1.IngressList{}

		// Search for ingresses in the application's namespace first
		listOptions := &client.ListOptions{Namespace: app.GetNamespace()}
		err = kube.List(ctx, ingressList, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list ingresses in namespace %s: %s", app.GetNamespace(), err)
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
			if matchedIngress.Namespace != app.GetNamespace() {
				if err = checkAppForAllowedRedirectAnnotation(ctx, kube, matchedIngress); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
