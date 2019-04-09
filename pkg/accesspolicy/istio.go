package accesspolicy

import (
	"github.com/fatih/structs"
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio "istio.io/api/rbac/v1alpha1"
	istio_crd "istio.io/istio/pilot/pkg/config/kube/crd"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const IstioAPIVersion = "v1alpha1"

func getDefaultServiceRole() *istio.ServiceRole {
	return &istio.ServiceRole{
		Rules: []*istio.AccessRule{
			{
				Services: []string{""},
				Methods:  []string{"*"},
			},
		},
	}
}

func getDefaultServiceRoleBinding(appName string) *istio.ServiceRoleBinding {
	return &istio.ServiceRoleBinding{
		Subjects: []*istio.Subject{{User: ""}},
		RoleRef: &istio.RoleRef{
			Kind: "RoleRef",
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
			ObjectMeta: k8s_meta.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
				Labels: map[string]string{
					"team": app.Labels["team"],
				},
			},
			Spec: structs.Map(getDefaultServiceRole()),
		}, &istio_crd.ServiceRoleBinding{
			TypeMeta: k8s_meta.TypeMeta{
				Kind:       "ServiceRoleBinding",
				APIVersion: IstioAPIVersion,
			},
			ObjectMeta: k8s_meta.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
				Labels: map[string]string{
					"team": app.Labels["team"],
				},
			},
			Spec: structs.Map(getDefaultServiceRoleBinding(app.Name)),
		},
	}
}
