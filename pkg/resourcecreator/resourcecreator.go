// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Create takes an Application resource and returns a slice of Kubernetes resources.
func Create(app *nais.Application) ([]runtime.Object, error) {
	objects := []runtime.Object{
		Service(app),
		Deployment(app),
		ServiceAccount(app),
		HorizontalPodAutoscaler(app),
	}

	ingress, err := Ingress(app)
	if err != nil {
		return nil, fmt.Errorf("while creating ingress: %s", err)
	}
	objects = append(objects, ingress)

	return objects, nil
}

func int32p(i int32) *int32 {
	return &i
}
