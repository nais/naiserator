package resourcecreator

import (
	"fmt"
	istio_crd "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const IstioAPIVersion = "v1alpha1"


func getServicePath(rule nais.AccessPolicyGressRule) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", rule.Application, rule.Namespace)
}

// If Services is set to [“*”], it refers to all services in the namespace defined in metadata.
func getAccessRules(rules []nais.AccessPolicyGressRule, allowAll bool) (accessRules []*istio_crd.AccessRule) {
	services := []string{}

	if allowAll {
		services = []string{"*"}
	} else {
		for _, gress := range rules {
			services = append(services, getServicePath(gress))
		}
	}

	return []*istio_crd.AccessRule{{
		Services: services,
		Methods: []string{"*"},
	}}
}


func getServiceRoleSpec(app *nais.Application) istio_crd.ServiceRoleSpec {
	return istio_crd.ServiceRoleSpec{
		Rules: getAccessRules(app.Spec.AccessPolicy.Ingress.Rules, app.Spec.AccessPolicy.Ingress.AllowAll),
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

func Istio(app *nais.Application) []runtime.Object {
	if len(app.Spec.AccessPolicy.Ingress.Rules) == 0 {
		return []runtime.Object{}
	}

	return []runtime.Object{
		&istio_crd.ServiceRole{
			TypeMeta: k8s_meta.TypeMeta{
				Kind:       "ServiceRole",
				APIVersion: IstioAPIVersion,
			},
			ObjectMeta: app.CreateObjectMeta(),
			Spec:       getServiceRoleSpec(app),
		}, &istio_crd.ServiceRoleBinding{
			TypeMeta: k8s_meta.TypeMeta{
				Kind:       "ServiceRoleBinding",
				APIVersion: IstioAPIVersion,
			},
			ObjectMeta: app.CreateObjectMeta(),
			Spec:       getDefaultServiceRoleBinding(app.Name, app.Namespace),
		},
	}
}


