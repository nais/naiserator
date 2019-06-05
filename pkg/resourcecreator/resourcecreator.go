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
func Create(app *nais.Application, resourceOptions ResourceOptions) ([]runtime.Object, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	objects := []runtime.Object{
		Service(app),
		ServiceAccount(app),
		HorizontalPodAutoscaler(app),
	}

	if resourceOptions.AccessPolicy {
		objects = append(objects, NetworkPolicy(app))
		objects = append(objects, VirtualService(app))

		serviceRole := ServiceRole(app)
		if serviceRole != nil {
			objects = append(objects, serviceRole)
		}
		serviceRoleBinding := ServiceRoleBinding(app)
		if serviceRoleBinding != nil {
			objects = append(objects, serviceRoleBinding)
		}
	}

	deployment, err := Deployment(app, resourceOptions)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	objects = append(objects, deployment)

	ingress, err := Ingress(app)
	if err != nil {
		return nil, fmt.Errorf("while creating ingress: %s", err)
	}
	if ingress != nil {
		// the application might have no ingresses, in which case nil will be returned.
		objects = append(objects, ingress)
	}

	return objects, nil
}

func int32p(i int32) *int32 {
	return &i
}
