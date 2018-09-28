// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Create takes an Application resource and returns a slice of Kubernetes resources.
func Create(app *nais.Application) []runtime.Object {
	return []runtime.Object{
		Service(app),
		Deployment(app),
		ServiceAccount(app),
		HorizontalPodAutoscaler(app),
		Ingress(app),
	}
}

func int32p(i int32) *int32 {
	return &i
}
