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
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ops := ResourceOperations{
		{Service(app), OperationCreateOrUpdate},
		{ServiceAccount(app, resourceOptions), OperationCreateIfNotExists},
		{HorizontalPodAutoscaler(app), OperationCreateOrUpdate},
	}

	if app.Spec.LeaderElection {
		leRole := LeaderElectionRole(app)
		leRoleBinding := LeaderElectionRoleBinding(app)
		ops = append(ops, ResourceOperation{leRole, OperationCreateOrUpdate})
		ops = append(ops, ResourceOperation{leRoleBinding, OperationCreateOrRecreate})
	}

	if app.Spec.AccessPolicy != nil {
		jwker := Jwker(app)
		ops = append(ops, ResourceOperation{jwker, OperationCreateOrUpdate})
	}

	if len(resourceOptions.GoogleProjectId) > 0 && app.Spec.GCP != nil {
		googleServiceAccount := GoogleIAMServiceAccount(app, resourceOptions.GoogleProjectId)
		googleServiceAccountBinding := GoogleIAMPolicy(app, &googleServiceAccount, resourceOptions.GoogleProjectId)
		ops = append(ops, ResourceOperation{&googleServiceAccount, OperationCreateOrUpdate})
		ops = append(ops, ResourceOperation{&googleServiceAccountBinding, OperationCreateOrUpdate})

		if app.Spec.GCP.Buckets != nil && len(app.Spec.GCP.Buckets) > 0 {
			for _, b := range app.Spec.GCP.Buckets {
				bucket := GoogleStorageBucket(app, b)
				ops = append(ops, ResourceOperation{bucket, OperationCreateIfNotExists})

				bucketAccessControl := GoogleStorageBucketAccessControl(app, bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
				ops = append(ops, ResourceOperation{bucketAccessControl, OperationCreateOrUpdate})
			}
		}

		if app.Spec.GCP.SqlInstances != nil {
			vars := make(map[string]string)

			for i, sqlInstance := range app.Spec.GCP.SqlInstances {
				if i > 0 {
					return nil, fmt.Errorf("only one sql instance is supported")
				}

				// TODO: name defaulting will break with more than one instance
				sqlInstance, err := CloudSqlInstanceWithDefaults(sqlInstance, app.Name)
				if err != nil {
					return nil, err
				}

				instance := GoogleSqlInstance(app, sqlInstance, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, ResourceOperation{instance, OperationCreateOrUpdate})

				key, err := util.Keygen(32)
				if err != nil {
					return nil, fmt.Errorf("unable to generate secret for sql user: %s", err)
				}
				username := instance.Name
				password := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(key)

				iamPolicyMember := SqlInstanceIamPolicyMember(app, sqlInstance.Name, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, ResourceOperation{iamPolicyMember, OperationCreateIfNotExists})

				for _, db := range sqlInstance.Databases {
					googledb := GoogleSQLDatabase(app, db, sqlInstance, resourceOptions.GoogleTeamProjectId)
					ops = append(ops, ResourceOperation{googledb, OperationCreateIfNotExists})
					env := GoogleSQLEnvVars(&db, instance.Name, username, password)
					for k, v := range env {
						vars[k] = v
					}
				}

				// FIXME: only works when there is one sql instance
				secretKeyRefEnvName, err := firstKeyWithSuffix(vars, googleSQLPasswordSuffix)
				if err != nil {
					return nil, fmt.Errorf("unable to assign sql password: %s", err)
				}
				sqlUser := GoogleSqlUser(app, instance, secretKeyRefEnvName, sqlInstance.CascadingDelete, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, ResourceOperation{sqlUser, OperationCreateIfNotExists})

				// FIXME: take into account when refactoring default values
				app.Spec.GCP.SqlInstances[i].Name = sqlInstance.Name
			}

			secret := OpaqueSecret(app, GoogleSQLSecretName(app), vars)
			ops = append(ops, ResourceOperation{secret, OperationCreateIfNotExists})
		}
	}

	if resourceOptions.AccessPolicy {
		ops = append(ops, ResourceOperation{NetworkPolicy(app, resourceOptions.AccessPolicyNotAllowedCIDRs), OperationCreateOrUpdate})
		vses, err := VirtualServices(app)

		if err != nil {
			return nil, fmt.Errorf("unable to create VirtualServices: %s", err)
		}

		for _, vs := range vses {
			ops = append(ops, ResourceOperation{vs, OperationCreateOrUpdate})
		}

		authorizationPolicy := AuthorizationPolicy(app)
		if authorizationPolicy != nil {
			ops = append(ops, ResourceOperation{authorizationPolicy, OperationCreateOrUpdate})
		}

		serviceEntries := ServiceEntries(app)
		for _, serviceEntry := range serviceEntries {
			ops = append(ops, ResourceOperation{serviceEntry, OperationCreateOrUpdate})
		}

	} else {

		ingress, err := Ingress(app)
		if err != nil {
			return nil, fmt.Errorf("while creating ingress: %s", err)
		}

		if ingress != nil {
			ops = append(ops, ResourceOperation{ingress, OperationCreateOrUpdate})
		}
	}

	deployment, err := Deployment(app, resourceOptions)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	ops = append(ops, ResourceOperation{deployment, OperationCreateOrUpdate})

	return ops, nil
}

func intp(i int) *int {
	return &i
}

func int32p(i int32) *int32 {
	return &i
}

func setAnnotation(resource v1.ObjectMetaAccessor, key, value string) {
	m := resource.GetObjectMeta().GetAnnotations()
	m[key] = value
	resource.GetObjectMeta().SetAnnotations(m)
}
