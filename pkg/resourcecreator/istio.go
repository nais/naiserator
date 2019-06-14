package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio_crd "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const IstioAPIVersion = "v1alpha1"

func getServicePath(rule nais.AccessPolicyGressRule, appNamespace string) string {
	namespace := rule.Namespace
	if len(namespace) == 0 {
		namespace = appNamespace
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", rule.Application, namespace)
}

// If Services is set to [“*”], it refers to all services in the namespace defined in metadata.
func getAccessRules(rules []nais.AccessPolicyGressRule, allowAll bool, appNamespace string) (accessRules []*istio_crd.AccessRule) {
	services := []string{}

	if allowAll {
		services = []string{"*"}
	}
	for _, gress := range rules {
		services = append(services, getServicePath(gress, appNamespace))
	}

	return []*istio_crd.AccessRule{{
		Services: services,
		Methods:  []string{"*"},
	}}
}

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

	if !app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) == 0 {
		return nil, nil
	}

	rules := app.Spec.AccessPolicy.Ingress.Rules
	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, nais.AccessPolicyGressRule{
			Namespace:   "istio-system",
			Application: "istio-ingressgateway-service-account",
		})
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
				Name: app.Namespace,
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

	rules := app.Spec.AccessPolicy.Ingress.Rules
	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, nais.AccessPolicyGressRule{
			Namespace:   "istio-system",
			Application: "istio-ingressgateway",
		})
	}

	return &istio_crd.ServiceRole{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRole",
			APIVersion: IstioAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio_crd.ServiceRoleSpec{
			Rules: getAccessRules(rules, app.Spec.AccessPolicy.Ingress.AllowAll, app.Namespace),
		},
	}, nil
}
