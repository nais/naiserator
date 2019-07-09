package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio_crd "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			APIVersion: IstioRBACAPIVersion,
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

func ServiceRoleBindingPrometheus(app *nais.Application) (serviceRoleBindingPrometheus *istio_crd.ServiceRoleBinding) {
	if !app.Spec.Prometheus.Enabled {
		return nil
	}

	name := fmt.Sprintf("%s-prometheus", app.Name)

	return &istio_crd.ServiceRoleBinding{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRoleBinding",
			APIVersion: IstioRBACAPIVersion,
		},
		ObjectMeta: app.CreateObjectMetaWithName(name),
		Spec: istio_crd.ServiceRoleBindingSpec{
			Subjects: []*istio_crd.Subject{
				{
					User: fmt.Sprintf("cluster.local/ns/%s/sa/%s", IstioNamespace, IstioPrometheusServiceAccount),
				},
			},
			RoleRef: &istio_crd.RoleRef{
				Kind: "ServiceRole",
				Name: name,
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
			APIVersion: IstioRBACAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio_crd.ServiceRoleSpec{
			Rules: []*istio_crd.AccessRule{
				{
					Methods:  []string{"*"},
					Services: []string{servicePath},
					Paths:    []string{"*"},
				},
			},
		},
	}
}

func ServiceRolePrometheus(app *nais.Application) (serviceRolePrometheus *istio_crd.ServiceRole) {
	if !app.Spec.Prometheus.Enabled {
		return nil
	}

	name := fmt.Sprintf("%s-prometheus", app.Name)

	servicePath := fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace)

	return &istio_crd.ServiceRole{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRole",
			APIVersion: IstioRBACAPIVersion,
		},
		ObjectMeta: app.CreateObjectMetaWithName(name),
		Spec: istio_crd.ServiceRoleSpec{
			Rules: []*istio_crd.AccessRule{
				{
					Methods:  []string{"GET"},
					Services: []string{servicePath},
					Paths:    []string{app.Spec.Prometheus.Path},
				},
			},
		},
	}
}
