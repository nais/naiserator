// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
)

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais.Application, resourceOptions ResourceOptions) (ResourceOperations, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	objects := ResourceOperations{
		{Service(app), OperationCreateOrUpdate},
		{ServiceAccount(app), OperationCreateOrUpdate},
		{HorizontalPodAutoscaler(app), OperationCreateOrUpdate},
	}

	if app.Spec.LeaderElection {
		objects = append(objects, ResourceOperation{LeaderElectionRole(app), OperationCreateOrUpdate})
		objects = append(objects, ResourceOperation{LeaderElectionRoleBinding(app), OperationCreateOrRecreate})
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

		// Kubernetes doesn't support ingress resources without any rules. This means we must
		// delete the old resource if it exists.
		operation := OperationCreateOrUpdate
		if len(app.Spec.Ingresses) == 0 {
			operation = OperationDeleteIfExists
		}

		objects = append(objects, ResourceOperation{ingress, operation})
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
