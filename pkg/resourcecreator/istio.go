package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio_crd "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IstioAPIVersion                   = "v1alpha1"
	IstioNamespace                    = "istio-system"
	IstioIngressGatewayServiceAccount = "istio-ingressgateway-service-account"
	IstioPrometheusServiceAccount     = "istio-prometheus-service-account"
)

func getServiceRoleBindingSubjects(rules []nais.AccessPolicyGressRule, appNamespace string, allowAll bool) (subjects []*istio_crd.Subject) {
	if allowAll {
		subjects = append(subjects, &istio_crd.Subject{User: "*"})
	}

	for _, rule := range rules {
		namespace := appNamespace
		if rule.Namespace != "" {
			namespace = rule.Namespace
		}
		subjects = append(subjects, &istio_crd.Subject{User: fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application)})
	}

	return
}

func ServiceRoleBinding(app *nais.Application) (*istio_crd.ServiceRoleBinding, error) {
	if app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		return nil, fmt.Errorf("cannot have access policy rules with allowAll = True")
	}

	rules := app.Spec.AccessPolicy.Ingress.Rules

	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, nais.AccessPolicyGressRule{
			Namespace:   IstioNamespace,
			Application: IstioIngressGatewayServiceAccount,
		})
	}

	if !app.Spec.AccessPolicy.Ingress.AllowAll && len(rules) == 0 {
		return nil, nil
	}

	return &istio_crd.ServiceRoleBinding{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRoleBinding",
			APIVersion: IstioAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio_crd.ServiceRoleBindingSpec{
			Subjects: getServiceRoleBindingSubjects(rules, app.Namespace, app.Spec.AccessPolicy.Ingress.AllowAll),
			RoleRef: &istio_crd.RoleRef{
				Kind: "ServiceRole",
				Name: app.Name,
			},
		},
	}, nil
}

func ServiceRoleBindingPrometheus(app *nais.Application) (serviceRoleBindingPrometheus *istio_crd.ServiceRoleBinding, err error) {
	if !app.Spec.Prometheus.Enabled {
		return nil, nil
	}

	name := fmt.Sprintf("%s-prometheus", app.Name)

	return &istio_crd.ServiceRoleBinding{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRoleBinding",
			APIVersion: IstioAPIVersion,
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
	}, nil
}

func ServiceRole(app *nais.Application) (*istio_crd.ServiceRole, error) {
	if app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		return nil, fmt.Errorf("cannot have access policy rules with allowAll = True")
	}

	if !app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) == 0 {
		return nil, nil
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
					Paths:    []string{"*"},
				},
			},
		},
	}, nil
}

func ServiceRolePrometheus(app *nais.Application) (serviceRolePrometheus *istio_crd.ServiceRole, err error) {
	if !app.Spec.Prometheus.Enabled {
		return nil, nil
	}

	name := fmt.Sprintf("%s-prometheus", app.Name)

	servicePath := fmt.Sprintf("%s.%s.svc.cluster.local", app.Name, app.Namespace)

	return &istio_crd.ServiceRole{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRole",
			APIVersion: IstioAPIVersion,
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
	}, nil
}
