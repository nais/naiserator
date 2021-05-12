// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	deployment "github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/idporten"
	"github.com/nais/naiserator/pkg/resourcecreator/ingress"
	"github.com/nais/naiserator/pkg/resourcecreator/jwker"
	"github.com/nais/naiserator/pkg/resourcecreator/kafka"
	"github.com/nais/naiserator/pkg/resourcecreator/leaderelection"
	"github.com/nais/naiserator/pkg/resourcecreator/linkerd"
	"github.com/nais/naiserator/pkg/resourcecreator/maskinporten"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/util"
)

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options) (resource.Operations, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ops := resource.Operations{}

	if resourceOptions.KafkaratorEnabled && app.Spec.Kafka != nil {
		var err error
		resourceOptions.KafkaratorSecretName, err = kafka.GenerateKafkaSecretName(app)
		if err != nil {
			return nil, err
		}
	}

	if len(resourceOptions.GoogleProjectId) > 0 {
		googleServiceAccount := google_iam.GoogleIAMServiceAccount(app, resourceOptions.GoogleProjectId)
		googleServiceAccountBinding := google_iam.GoogleIAMPolicy(app, &googleServiceAccount, resourceOptions.GoogleProjectId)
		ops = append(ops, resource.Operation{&googleServiceAccount, resource.OperationCreateOrUpdate})
		ops = append(ops, resource.Operation{&googleServiceAccountBinding, resource.OperationCreateOrUpdate})

		if app.Spec.GCP != nil && app.Spec.GCP.Buckets != nil && len(app.Spec.GCP.Buckets) > 0 {
			for _, b := range app.Spec.GCP.Buckets {
				bucket := google_storagebucket.GoogleStorageBucket(app, b)
				ops = append(ops, resource.Operation{bucket, resource.OperationCreateIfNotExists})

				bucketAccessControl := google_storagebucket.GoogleStorageBucketAccessControl(app, bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
				ops = append(ops, resource.Operation{bucketAccessControl, resource.OperationCreateOrUpdate})

				iamPolicyMember := google_storagebucket.StorageBucketIamPolicyMember(app, bucket, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, resource.Operation{iamPolicyMember, resource.OperationCreateIfNotExists})
			}
		}

		if app.Spec.GCP != nil && app.Spec.GCP.SqlInstances != nil {

			for i, sqlInstance := range app.Spec.GCP.SqlInstances {
				if i > 0 {
					return nil, fmt.Errorf("only one sql instance is supported")
				}

				// TODO: name defaulting will break with more than one instance
				sqlInstance, err := google_sql.CloudSqlInstanceWithDefaults(sqlInstance, app.Name)
				if err != nil {
					return nil, err
				}

				instance := google_sql.GoogleSqlInstance(app, sqlInstance, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, resource.Operation{instance, resource.OperationCreateOrUpdate})

				iamPolicyMember := google_sql.SqlInstanceIamPolicyMember(app, sqlInstance.Name, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, resource.Operation{iamPolicyMember, resource.OperationCreateIfNotExists})

				for _, db := range sqlInstance.Databases {
					sqlUsers := google_sql.MergeAndFilterSQLUsers(db.Users, instance.Name)

					googledb := google_sql.GoogleSQLDatabase(app, db, sqlInstance, resourceOptions.GoogleTeamProjectId)
					ops = append(ops, resource.Operation{googledb, resource.OperationCreateIfNotExists})

					for _, user := range sqlUsers {
						vars := make(map[string]string)

						googleSqlUser := google_sql.SetupNewGoogleSqlUser(user.Name, &db, instance)

						password, err := util.GeneratePassword()
						if err != nil {
							return nil, err
						}

						env := googleSqlUser.CreateUserEnvVars(password)
						vars = google_sql.MapEnvToVars(env, vars)

						secretKeyRefEnvName, err := googleSqlUser.KeyWithSuffixMatchingUser(vars, google_sql.GoogleSQLPasswordSuffix)
						if err != nil {
							return nil, fmt.Errorf("unable to assign sql password: %s", err)
						}

						sqlUser, err := googleSqlUser.Create(app, secretKeyRefEnvName, sqlInstance.CascadingDelete, resourceOptions.GoogleTeamProjectId)
						if err != nil {
							return nil, fmt.Errorf("unable to create sql user: %s", err)
						}
						ops = append(ops, resource.Operation{sqlUser, resource.OperationCreateIfNotExists})

						scrt := secret.OpaqueSecret(app, google_sql.GoogleSQLSecretName(app, googleSqlUser.Instance.Name, googleSqlUser.Name), vars)
						ops = append(ops, resource.Operation{scrt, resource.OperationCreateIfNotExists})
					}
				}

				// FIXME: take into account when refactoring default values
				app.Spec.GCP.SqlInstances[i].Name = sqlInstance.Name
			}
		}

		if app.Spec.GCP != nil && app.Spec.GCP.Permissions != nil {
			for _, p := range app.Spec.GCP.Permissions {
				policy, err := google_iam.GoogleIAMPolicyMember(app, p, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
				if err != nil {
					return nil, fmt.Errorf("unable to create iampolicymember: %w", err)
				}
				ops = append(ops, resource.Operation{policy, resource.OperationCreateIfNotExists})
			}
		}
	}

	service.Create(app, &ops)
	serviceaccount.Create(app, resourceOptions, &ops)
	horizontalpodautoscaler.Create(app, &ops)
	dplt, err := deployment.Create(app, resourceOptions, &ops)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	err = azure.Create(app, resourceOptions, dplt, &ops)
	if err != nil {
		return nil, err
	}
	err = idporten.Create(app, resourceOptions, dplt, &ops)
	if err != nil {
		return nil, err
	}
	maskinporten.Create(app, resourceOptions, dplt, &ops)
	poddisruptionbudget.Create(app, &ops)
	jwker.Create(app, resourceOptions, dplt, &ops)
	leaderelection.Create(app, dplt, &ops)
	aiven.Elastic(app, dplt)
	linkerd.Create(resourceOptions, dplt)
	networkpolicy.Create(app, resourceOptions, &ops)
	err = ingress.Create(app, resourceOptions, &ops)
	if err != nil {
		return nil, fmt.Errorf("while creating ingress: %s", err)
	}

	return ops, nil
}
