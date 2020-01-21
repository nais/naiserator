// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"encoding/base64"
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais.Application, resourceOptions ResourceOptions) (ResourceOperations, error) {
	var operation Operation

	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ops := ResourceOperations{
		{Service(app), OperationCreateOrUpdate},
		{ServiceAccount(app, resourceOptions), OperationCreateOrUpdate},
		{HorizontalPodAutoscaler(app), OperationCreateOrUpdate},
	}

	leRole := LeaderElectionRole(app)
	leRoleBinding := LeaderElectionRoleBinding(app)

	if app.Spec.LeaderElection {
		ops = append(ops, ResourceOperation{leRole, OperationCreateOrUpdate})
		ops = append(ops, ResourceOperation{leRoleBinding, OperationCreateOrRecreate})
	} else {
		ops = append(ops, ResourceOperation{leRole, OperationDeleteIfExists})
		ops = append(ops, ResourceOperation{leRoleBinding, OperationDeleteIfExists})
	}

	if len(resourceOptions.GoogleProjectId) > 0 && app.Spec.GCP != nil {
		// TODO: A service account will be required for all GCP related resources.
		// TODO: If implementing more features, move these two outside of the cloud storage check.
		googleServiceAccount := GoogleIAMServiceAccount(app)
		googleServiceAccountBinding := GoogleIAMPolicy(app, &googleServiceAccount, resourceOptions.GoogleProjectId)
		ops = append(ops, ResourceOperation{&googleServiceAccount, OperationCreateIfNotExists})
		ops = append(ops, ResourceOperation{&googleServiceAccountBinding, OperationCreateIfNotExists})

		if app.Spec.GCP.Buckets != nil && len(app.Spec.GCP.Buckets) > 0 {
			for _, b := range app.Spec.GCP.Buckets {
				bucket := GoogleStorageBucket(app, b)
				ops = append(ops, ResourceOperation{bucket, OperationCreateIfNotExists})

				bucketAccessControl := GoogleStorageBucketAccessControl(app, bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
				ops = append(ops, ResourceOperation{bucketAccessControl, OperationCreateOrUpdate})
			}
		}

		if app.Spec.GCP.SqlInstances != nil {
			for i, sqlInstance := range app.Spec.GCP.SqlInstances {
				if i > 0 {
					return nil, fmt.Errorf("only one sql instance is supported")
				}

				// TODO: name defaulting will break with more than one instance
				sqlInstance, err := CloudSqlInstanceWithDefaults(sqlInstance, app.Name)
				if err != nil {
					return nil, err
				}

				instance := GoogleSqlInstance(app, sqlInstance, resourceOptions.GoogleProjectId)
				ops = append(ops, ResourceOperation{instance, OperationCreateOrUpdate})

				iamPolicyMember := SqlInstanceIamPolicyMember(app, sqlInstance.Name, resourceOptions.GoogleProjectId)
				ops = append(ops, ResourceOperation{iamPolicyMember, OperationCreateIfNotExists})

				for _, db := range GoogleSqlDatabases(app, sqlInstance, resourceOptions.GoogleProjectId) {
					ops = append(ops, ResourceOperation{db, OperationCreateIfNotExists})
				}

				key, err := util.Keygen(32)
				if err != nil {
					return nil, fmt.Errorf("unable to generate secret for sql user: %s", err)
				}
				password := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(key)

				secret := OpaqueSecret(app, GCPSqlInstanceSecretName(instance.Name), GoogleSqlUserEnvVars(instance.Name, password))
				ops = append(ops, ResourceOperation{secret, OperationCreateIfNotExists})

				sqlUser := GoogleSqlUser(app, instance.Name, sqlInstance.CascadingDelete, resourceOptions.GoogleProjectId)
				ops = append(ops, ResourceOperation{sqlUser, OperationCreateIfNotExists})

				// FIXME: take into account when refactoring default values
				app.Spec.GCP.SqlInstances[i].Name = sqlInstance.Name
			}
		}
	}

	if resourceOptions.AccessPolicy {
		ops = append(ops, ResourceOperation{NetworkPolicy(app, resourceOptions.AccessPolicyNotAllowedCIDRs), OperationCreateOrUpdate})
		vses, err := VirtualServices(app)

		if err != nil {
			return nil, fmt.Errorf("unable to create VirtualServices: %s", err)
		}

		operation = OperationCreateOrUpdate
		if len(app.Spec.Ingresses) == 0 {
			operation = OperationDeleteIfExists
		}

		for _, vs := range vses {
			ops = append(ops, ResourceOperation{vs, operation})
		}

		// Applies to ServiceRoles and ServiceRoleBindings
		operation = OperationCreateOrUpdate
		if len(app.Spec.AccessPolicy.Inbound.Rules) == 0 && len(app.Spec.Ingresses) == 0 {
			operation = OperationDeleteIfExists
		}

		serviceRole := ServiceRole(app)
		if serviceRole != nil {
			ops = append(ops, ResourceOperation{serviceRole, operation})
		}

		serviceRoleBinding := ServiceRoleBinding(app)
		if serviceRoleBinding != nil {
			ops = append(ops, ResourceOperation{serviceRoleBinding, operation})
		}

		serviceRolePrometheus := ServiceRolePrometheus(app)
		if serviceRolePrometheus != nil {
			ops = append(ops, ResourceOperation{serviceRolePrometheus, OperationCreateOrUpdate})
		}

		serviceRoleBindingPrometheus := ServiceRoleBindingPrometheus(app)
		operation = OperationCreateOrUpdate
		if !app.Spec.Prometheus.Enabled {
			operation = OperationDeleteIfExists
		}

		if serviceRoleBindingPrometheus != nil {
			ops = append(ops, ResourceOperation{serviceRoleBindingPrometheus, operation})
		}

		serviceEntry := ServiceEntry(app)
		operation = OperationCreateOrUpdate
		if len(app.Spec.AccessPolicy.Outbound.External) == 0 {
			operation = OperationDeleteIfExists
		}
		if serviceEntry != nil {
			ops = append(ops, ResourceOperation{serviceEntry, operation})
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

		ops = append(ops, ResourceOperation{ingress, operation})
	}

	deployment, err := Deployment(app, resourceOptions)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	ops = append(ops, ResourceOperation{deployment, OperationCreateOrUpdate})

	return ops, nil
}

func int32p(i int32) *int32 {
	return &i
}

func setAnnotation(resource v1.ObjectMetaAccessor, key, value string) {
	m := resource.GetObjectMeta().GetAnnotations()
	m[key] = value
	resource.GetObjectMeta().SetAnnotations(m)
}
