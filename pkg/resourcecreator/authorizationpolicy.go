package resourcecreator

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	security_istio_io_v1beta1 "github.com/nais/liberator/pkg/apis/security.istio.io/v1beta1"
	"github.com/nais/naiserator/pkg/util"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const PrometheusServiceAccountPrincipal = "cluster.local/ns/istio-system/sa/prometheus"

func AuthorizationPolicy(app *nais.Application, options ResourceOptions) (*security_istio_io_v1beta1.AuthorizationPolicy, error) {
	var rules []*security_istio_io_v1beta1.Rule

	// TODO: This is the old ingress-gateway, need this until it is removed from the clusters
	gateways := []string{"istio-system/istio-ingressgateway"}
	for _, ingress := range app.Spec.Ingresses {
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

		// Avoid duplicate gateways, as this will look messy and lead to unnecessary app synchronizations
		for _, gateway := range ResolveGateway(parsedUrl.Host, options.GatewayMappings) {
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

	return &security_istio_io_v1beta1.AuthorizationPolicy{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "AuthorizationPolicy",
			APIVersion: IstioAuthorizationPolicyVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: security_istio_io_v1beta1.AuthorizationPolicySpec{
			Selector: &security_istio_io_v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": app.Name},
			},
			Rules: rules,
		},
	}, nil
}

func ingressGatewayRule(gateways []string) *security_istio_io_v1beta1.Rule {
	var principals []string
	for _, gateway := range gateways {
		parts := strings.Split(gateway, "/")
		namespace, serviceAccount := parts[0], parts[1]
		principals = append(principals, fmt.Sprintf("cluster.local/ns/%s/sa/%s-service-account", namespace, serviceAccount))
	}

	// Avoid unnecessary synchronization if order changes
	sort.Strings(principals)

	return &security_istio_io_v1beta1.Rule{
		From: []*security_istio_io_v1beta1.Rule_From{
			{
				Source: &security_istio_io_v1beta1.Source{
					Principals: principals,
				},
			},
		},
		To: []*security_istio_io_v1beta1.Rule_To{
			{
				Operation: &security_istio_io_v1beta1.Operation{
					Paths: []string{"*"},
				},
			},
		},
	}
}

func accessPolicyRules(app *nais.Application, options ResourceOptions) *security_istio_io_v1beta1.Rule {
	principals := principals(app, options)
	if len(principals) < 1 {
		return nil
	}
	return &security_istio_io_v1beta1.Rule{
		From: []*security_istio_io_v1beta1.Rule_From{
			{
				Source: &security_istio_io_v1beta1.Source{
					Principals: principals,
				},
			},
		},
	}
}

func prometheusRule(app *nais.Application) *security_istio_io_v1beta1.Rule {
	port := app.Spec.Prometheus.Port
	if len(port) == 0 {
		port = strconv.Itoa(app.Spec.Port)
	}
	return &security_istio_io_v1beta1.Rule{
		From: []*security_istio_io_v1beta1.Rule_From{
			{
				Source: &security_istio_io_v1beta1.Source{
					Principals: []string{PrometheusServiceAccountPrincipal},
				},
			},
		},
		To: []*security_istio_io_v1beta1.Rule_To{
			{
				Operation: &security_istio_io_v1beta1.Operation{
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
