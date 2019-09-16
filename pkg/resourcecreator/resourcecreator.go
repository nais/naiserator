// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Operation defines what should be done with a resource.
type Operation int

const (
	OperationCreateOrUpdate Operation = iota
	OperationCreateOrRecreate
	OperationDeleteIfExists
)

// ResourceOperation is the combination of a Kubernetes resource and what operation to perform on it.
type ResourceOperation struct {
	Resource  runtime.Object
	Operation Operation
}

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais.Application, resourceOptions ResourceOptions) ([]ResourceOperation, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	objects := []ResourceOperation{
		{Service(app), OperationCreateOrUpdate},
		{ServiceAccount(app), OperationCreateOrUpdate},
		{HorizontalPodAutoscaler(app), OperationCreateOrUpdate},
	}

	if app.Spec.LeaderElection {
		objects = append(objects, ResourceOperation{LeaderElectionRole(app), OperationCreateOrUpdate})
		objects = append(objects, ResourceOperation{LeaderElectionRoleBinding(app), OperationCreateOrUpdate})
	}

	if resourceOptions.AccessPolicy {
		objects = append(objects, ResourceOperation{NetworkPolicy(app), OperationCreateOrUpdate})

		vses, err := VirtualServices(app)

		if err != nil {
			return nil, fmt.Errorf("unable to create VirtualServices: %s", err)
		}

		for _, vs := range vses {
			objects = append(objects, ResourceOperation{vs, OperationCreateOrUpdate})
		}

		serviceRole := ServiceRole(app)
		if serviceRole != nil {
			objects = append(objects, ResourceOperation{serviceRole, OperationCreateOrUpdate})
		}

		serviceRoleBinding := ServiceRoleBinding(app)
		if serviceRoleBinding != nil {
			objects = append(objects, ResourceOperation{serviceRoleBinding, OperationCreateOrUpdate})
		}

		serviceRolePrometheus := ServiceRolePrometheus(app)
		if serviceRolePrometheus != nil {
			objects = append(objects, ResourceOperation{serviceRolePrometheus, OperationCreateOrUpdate})
		}

		serviceRoleBindingPrometheus := ServiceRoleBindingPrometheus(app)
		if serviceRoleBindingPrometheus != nil {
			objects = append(objects, ResourceOperation{serviceRoleBindingPrometheus, OperationCreateOrUpdate})
		}

		serviceEntry := ServiceEntry(app)
		if serviceEntry != nil {
			objects = append(objects, ResourceOperation{serviceEntry, OperationCreateOrUpdate})
		}

	} else {

		ingress, err := Ingress(app)
		if err != nil {
			return nil, fmt.Errorf("while creating ingress: %s", err)
		}
		if ingress != nil {
			// the application might have no ingresses, in which case nil will be returned.
			objects = append(objects, ResourceOperation{ingress, OperationCreateOrUpdate})
		}
	}

	deployment, err := Deployment(app, resourceOptions)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	objects = append(objects, ResourceOperation{deployment, OperationCreateOrUpdate})

	return objects, nil
}

func int32p(i int32) *int32 {
	return &i
}
