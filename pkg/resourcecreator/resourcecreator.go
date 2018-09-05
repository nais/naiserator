package resourcecreator

import (
	nais "github.com/nais/naiserator/api/types/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)


func GetResources(app *nais.Application) ([]runtime.Object, error) {
	return []runtime.Object{
		getService(app),
		getDeployment(app),
	}, nil
}

func getObjectMeta(app *nais.Application) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      app.Name,
		Namespace: app.Namespace,
		Labels: map[string]string{
			"app":  app.Name,
			"team": app.Spec.Team,
		},
		OwnerReferences: getOwnerReferences(app),
	}
}

func getOwnerReferences(app *nais.Application) []metav1.OwnerReference {
	return []metav1.OwnerReference{app.GetOwnerReference()}
}

func int32p(i int32) *int32 {
	return &i
}
