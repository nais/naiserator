package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Create(app *nais.Application) ([]runtime.Object, error) {
	return []runtime.Object{
		service(app),
		deployment(app),
		serviceAccount(app),
	}, nil
}

func int32p(i int32) *int32 {
	return &i
}
