package resourcecreator

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "istio.io/api/security/v1beta1"
	"istio.io/api/type/v1beta1"
	istio_security_client "istio.io/client-go/pkg/apis/security/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const PrometheusServiceAccountPrincipal = "cluster.local/ns/istio-system/sa/prometheus"

func AuthorizationPolicy(app *nais.Application, options ResourceOptions) (*istio_security_client.AuthorizationPolicy, error) {
	var rules []*istio.Rule

	// TODO: This is the old ingress-gateway, need this until it is removed from the clusters
	gateways := []string{"istio-system/istio-ingressgateway"}
	for _, ingress := range app.Spec.Ingresses {
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

		// Avoid duplicate gateways, as this will look messy and lead to unnecessary app synchronizations
		for _, gateway := range ResolveGateway(*parsedUrl, options.GatewayMappings) {
			found := false
			for _, existingGateway := range gateways {
				if gateway == existingGateway {
					found = true
				}
			}

			if !found {
				gateways = append(gateways, gateway)
			}
		}

	}

	if app.Spec.Prometheus.Enabled {
		rules = append(rules, prometheusRule(app))
	}

	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, ingressGatewayRule(gateways))
	}

	if len(app.Spec.AccessPolicy.Inbound.Rules) > 0 {
		accessPolicyRules := accessPolicyRules(app, options)
		if accessPolicyRules != nil {
			rules = append(rules, accessPolicyRules)
		}
	}

	if len(rules) == 0 {
		return nil, nil
	}

	return &istio_security_client.AuthorizationPolicy{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "AuthorizationPolicy",
			APIVersion: IstioAuthorizationPolicyVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": app.Name},
			},
			Rules: rules,
		},
	}, nil
}

func ingressGatewayRule(gateways []string) *istio.Rule {
	var principals []string
	for _, gateway := range gateways {
		parts := strings.Split(gateway, "/")
		namespace, serviceAccount := parts[0], parts[1]
		principals = append(principals, fmt.Sprintf("cluster.local/ns/%s/sa/%s-service-account", namespace, serviceAccount))
	}

	// Avoid unnecessary synchronization if order changes
	sort.Strings(principals)

	return &istio.Rule{
		From: []*istio.Rule_From{
			{
				Source: &istio.Source{
					Principals: principals,
				},
			},
		},
		To: []*istio.Rule_To{
			{
				Operation: &istio.Operation{
					Paths: []string{"*"},
				},
			},
		},
	}
}

func accessPolicyRules(app *nais.Application, options ResourceOptions) *istio.Rule {
	principals := principals(app, options)
	if len(principals) < 1 {
		return nil
	}
	return &istio.Rule{
		From: []*istio.Rule_From{
			{
				Source: &istio.Source{
					Principals: principals,
				},
			},
		},
	}
}

func prometheusRule(app *nais.Application) *istio.Rule {
	port := app.Spec.Prometheus.Port
	if len(port) == 0 {
		port = strconv.Itoa(app.Spec.Port)
	}
	return &istio.Rule{
		From: []*istio.Rule_From{
			{
				Source: &istio.Source{
					Principals: []string{PrometheusServiceAccountPrincipal},
				},
			},
		},
		To: []*istio.Rule_To{
			{
				Operation: &istio.Operation{
					Paths:   []string{app.Spec.Prometheus.Path},
					Ports:   []string{port},
					Methods: []string{"GET"},
				},
			},
		},
	}
}

func principals(app *nais.Application, options ResourceOptions) []string {
	var principals []string

	for _, rule := range app.Spec.AccessPolicy.Inbound.Rules {
		var namespace string
		// non-local access policy rules do not result in istio policies
		if !rule.MatchesCluster(options.ClusterName) {
			continue
		}
		if rule.Namespace == "" {
			namespace = app.Namespace
		} else {
			namespace = rule.Namespace
		}
		tmp := fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application)
		principals = append(principals, tmp)
	}
	return principals
}
