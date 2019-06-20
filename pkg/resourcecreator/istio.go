package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio_crd "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IstioAPIVersion =  "v1alpha1"
	IstioNamespace = "istio-system"
	IstioIngressGatewayServiceAccount = "istio-ingressgateway-service-account"
	IstioPrometheusServiceAccount = "default"
)

func getServiceRoleBindingSubjects(rules []nais.AccessPolicyGressRule, appNamespace string) (subjects []*istio_crd.Subject) {
	for _, rule := range rules {
		namespace := appNamespace
		if rule.Namespace != "" {
			namespace = rule.Namespace
		}
		subjects = append(subjects, &istio_crd.Subject{User: fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application)})
	}

	return
}

func ServiceRoleBinding(app *nais.Application) *istio_crd.ServiceRoleBinding {
	rules := app.Spec.AccessPolicy.Inbound.Rules
	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, nais.AccessPolicyGressRule{
			Namespace:   IstioNamespace,
			Application: IstioIngressGatewayServiceAccount,
		})
	}

	if app.Spec.Prometheus.Enabled {
		rules = append(rules, nais.AccessPolicyGressRule{
			Namespace:   IstioNamespace,
			Application: IstioPrometheusServiceAccount,
		})
	}

	if len(rules) == 0 && len(app.Spec.Ingresses) == 0 {
		return nil
	}

	return &istio_crd.ServiceRoleBinding{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRoleBinding",
			APIVersion: IstioAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio_crd.ServiceRoleBindingSpec{
			Subjects: getServiceRoleBindingSubjects(rules, app.Namespace),
			RoleRef: &istio_crd.RoleRef{
				Kind: "ServiceRole",
				Name: app.Name,
			},
		},
	}
}

func ServiceRole(app *nais.Application) *istio_crd.ServiceRole {
	if len(app.Spec.AccessPolicy.Inbound.Rules) == 0 && len(app.Spec.Ingresses) == 0 {
		return nil
	}

	servicePath := fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace)

	return &istio_crd.ServiceRole{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRole",
			APIVersion: IstioAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio_crd.ServiceRoleSpec{
			Rules: []*istio_crd.AccessRule{
				{
					Methods:  []string{"*"},
					Services: []string{servicePath},
				},
			},
		},
	}
}
