package accesspolicy

import (
	"fmt"
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio_crd "github.com/nais/naiserator/pkg/apis/istio/v1alpha1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const IstioAPIVersion = "v1alpha1"


func getServicePath(rule nais.AccessPolicyGressRule) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", rule.Application, rule.Namespace)
}

func getAccessRules(rules []nais.AccessPolicyGressRule) (accessRules []*istio_crd.AccessRule) {
	for _, gress := range rules {
		policy := istio_crd.AccessRule{
			Services: []string{getServicePath(gress)},
			Methods: []string{"*"},
		}

		accessRules = append(accessRules, &policy)
	}

	return
}


func getServiceRole(app *nais.Application) istio_crd.ServiceRoleSpec {
	return istio_crd.ServiceRoleSpec{
		Rules: getAccessRules(app.Spec.AccessPolicy.Ingress.Rules),
	}
}

func getDefaultServiceRoleBinding(appName string) istio_crd.ServiceRoleBindingSpec {
	return istio_crd.ServiceRoleBindingSpec{
		Subjects: []*istio_crd.Subject{{User: "*"}},
		RoleRef: &istio_crd.RoleRef{
			Kind: "ServiceRole",
			Name: appName,
		},
	}
}

func Istio(app *nais.Application) []runtime.Object {
	return []runtime.Object{
		&istio_crd.ServiceRole{
			TypeMeta: k8s_meta.TypeMeta{
				Kind:       "ServiceRole",
				APIVersion: IstioAPIVersion,
			},
			ObjectMeta: app.CreateObjectMeta(),
			Spec:       getServiceRole(app),
		}, &istio_crd.ServiceRoleBinding{
			TypeMeta: k8s_meta.TypeMeta{
				Kind:       "ServiceRoleBinding",
				APIVersion: IstioAPIVersion,
			},
			ObjectMeta: app.CreateObjectMeta(),
			Spec: getDefaultServiceRoleBinding(app.Name),
		},
	}
}
