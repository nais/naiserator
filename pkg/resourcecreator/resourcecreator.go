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
	var operation Operation

	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	objects := ResourceOperations{
		{Service(app), OperationCreateOrUpdate},
		{ServiceAccount(app), OperationCreateOrUpdate},
		{HorizontalPodAutoscaler(app), OperationCreateOrUpdate},
	}

	leRole := LeaderElectionRole(app)
	leRoleBinding := LeaderElectionRoleBinding(app)

	if app.Spec.LeaderElection {
		objects = append(objects, ResourceOperation{leRole, OperationCreateOrUpdate})
		objects = append(objects, ResourceOperation{leRoleBinding, OperationCreateOrRecreate})
	} else {
		objects = append(objects, ResourceOperation{leRole, OperationDeleteIfExists})
		objects = append(objects, ResourceOperation{leRoleBinding, OperationDeleteIfExists})
	}

	if resourceOptions.GoogleCluster {
		googleServiceAccount := GoogleServiceAccount(app)
		objects = append(objects, ResourceOperation{&googleServiceAccount, OperationCreateOrUpdate})
	}

	if resourceOptions.AccessPolicy {
		objects = append(objects, ResourceOperation{NetworkPolicy(app, resourceOptions.AccessPolicyNotAllowedCIDRs), OperationCreateOrUpdate})
		vses, err := VirtualServices(app)

		if err != nil {
			return nil, fmt.Errorf("unable to create VirtualServices: %s", err)
		}

		operation = OperationCreateOrUpdate
		if len(app.Spec.Ingresses) == 0 {
			operation = OperationDeleteIfExists
		}

		for _, vs := range vses {
			objects = append(objects, ResourceOperation{vs, operation})
		}

		serviceRole := ServiceRole(app)
		if serviceRole != nil {
			objects = append(objects, ResourceOperation{serviceRole, OperationCreateOrUpdate})
		}

		serviceRoleBinding := ServiceRoleBinding(app)
		operation = OperationCreateOrUpdate
		if len(app.Spec.AccessPolicy.Inbound.Rules) == 0 && len(app.Spec.Ingresses) == 0 {
			operation = OperationDeleteIfExists
		}

		if serviceRoleBinding != nil {
			objects = append(objects, ResourceOperation{serviceRoleBinding, operation})
		}

		serviceRolePrometheus := ServiceRolePrometheus(app)
		if serviceRolePrometheus != nil {
			objects = append(objects, ResourceOperation{serviceRolePrometheus, OperationCreateOrUpdate})
		}

		serviceRoleBindingPrometheus := ServiceRoleBindingPrometheus(app)
		operation = OperationCreateOrUpdate
		if app.Spec.Prometheus.Path == "" {
			operation = OperationDeleteIfExists
		}

		if serviceRoleBindingPrometheus != nil {
			objects = append(objects, ResourceOperation{serviceRoleBindingPrometheus, operation})
		}

		serviceEntry := ServiceEntry(app)
		operation = OperationCreateOrUpdate
		if len(app.Spec.AccessPolicy.Outbound.External) == 0 {
			operation = OperationDeleteIfExists
		}
		if serviceEntry != nil {
			objects = append(objects, ResourceOperation{serviceEntry, operation})
		}

	} else {

		ingress, err := Ingress(app)
		if err != nil {
			return nil, fmt.Errorf("while creating ingress: %s", err)
		}

		// Kubernetes doesn't support ingress resources without any rules. This means we must
		// delete the old resource if it exists.
		operation = OperationCreateOrUpdate
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
