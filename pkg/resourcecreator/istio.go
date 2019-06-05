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
	} else {
		for _, gress := range rules {
			services = append(services, getServicePath(gress, appNamespace))
		}
	}

	return []*istio_crd.AccessRule{{
		Services: services,
		Methods: []string{"*"},
	}}
}


func getServiceRoleSpec(app *nais.Application) istio_crd.ServiceRoleSpec {
	return istio_crd.ServiceRoleSpec{
		Rules: getAccessRules(app.Spec.AccessPolicy.Ingress.Rules, app.Spec.AccessPolicy.Ingress.AllowAll, app.Namespace),
	}
}

func getDefaultServiceRoleBinding(appName string, namespace string) istio_crd.ServiceRoleBindingSpec {
	return istio_crd.ServiceRoleBindingSpec{
		Subjects: []*istio_crd.Subject{{User: fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, appName)}},
		RoleRef: &istio_crd.RoleRef{
			Kind: "ServiceRole",
			Name: appName,
		},
	}
}

func ServiceRoleBinding(app *nais.Application) *istio_crd.ServiceRoleBinding {
	if !app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) == 0 {
		return nil
	}

	return &istio_crd.ServiceRoleBinding{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRoleBinding",
			APIVersion: IstioAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec:       getDefaultServiceRoleBinding(app.Name, app.Namespace),
	}

}

func ServiceRole(app *nais.Application) (*istio_crd.ServiceRole, error) {
	if !app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) > 0 {
		return nil, fmt.Errorf("cannot have ingress rules with allowAll = True")
	}

	if !app.Spec.AccessPolicy.Ingress.AllowAll && len(app.Spec.AccessPolicy.Ingress.Rules) == 0 {
		return nil, nil
	}

	return &istio_crd.ServiceRole{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "ServiceRole",
			APIVersion: IstioAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec:       getServiceRoleSpec(app),
	}, nil
}


