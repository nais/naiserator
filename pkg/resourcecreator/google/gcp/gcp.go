package gcp

import (
	"fmt"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
)

func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) error {
	if len(resourceOptions.GoogleProjectId) <= 0 {
		return nil
	}

	googleServiceAccount := google_iam.GoogleIAMServiceAccount(app, resourceOptions.GoogleProjectId)
	googleServiceAccountBinding := google_iam.GoogleIAMPolicy(app, &googleServiceAccount, resourceOptions.GoogleProjectId)
	*operations = append(*operations, resource.Operation{Resource: &googleServiceAccount, Operation: resource.OperationCreateOrUpdate})
	*operations = append(*operations, resource.Operation{Resource: &googleServiceAccountBinding, Operation: resource.OperationCreateOrUpdate})

	if app.Spec.GCP != nil {
		createBucket(app, resourceOptions, operations, googleServiceAccount)
		err := createSqlInstance(app, resourceOptions, deployment, operations)
		if err != nil {
			return err
		}
		err = createPermissions(app, resourceOptions, operations)
		if err != nil {
			return err
		}
	}

	return nil
}

func createPermissions(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, operations *resource.Operations) error {
	if app.Spec.GCP.Permissions != nil {
		for _, p := range app.Spec.GCP.Permissions {
			policy, err := google_iam.GoogleIAMPolicyMember(app, p, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
			if err != nil {
				return fmt.Errorf("unable to create iampolicymember: %w", err)
			}
			*operations = append(*operations, resource.Operation{Resource: policy, Operation: resource.OperationCreateIfNotExists})
		}
	}
	return nil
}

func createSqlInstance(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment, operations *resource.Operations) error {
	if app.Spec.GCP.SqlInstances != nil {
		for i, sqlInstance := range app.Spec.GCP.SqlInstances {
			if i > 0 {
				return fmt.Errorf("only one sql instance is supported")
			}

			// TODO: name defaulting will break with more than one instance
			sqlInstance, err := google_sql.CloudSqlInstanceWithDefaults(sqlInstance, app.Name)
			if err != nil {
				return err
			}

			instance := google_sql.GoogleSqlInstance(app, sqlInstance, resourceOptions.GoogleTeamProjectId)
			*operations = append(*operations, resource.Operation{Resource: instance, Operation: resource.OperationCreateOrUpdate})

			iamPolicyMember := google_sql.SqlInstanceIamPolicyMember(app, sqlInstance.Name, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
			*operations = append(*operations, resource.Operation{Resource: iamPolicyMember, Operation: resource.OperationCreateIfNotExists})

			for _, db := range sqlInstance.Databases {
				sqlUsers := google_sql.MergeAndFilterSQLUsers(db.Users, instance.Name)

				googledb := google_sql.GoogleSQLDatabase(app, db, sqlInstance, resourceOptions.GoogleTeamProjectId)
				*operations = append(*operations, resource.Operation{Resource: googledb, Operation: resource.OperationCreateIfNotExists})

				for _, user := range sqlUsers {
					vars := make(map[string]string)

					googleSqlUser := google_sql.SetupNewGoogleSqlUser(user.Name, &db, instance)

					password, err := util.GeneratePassword()
					if err != nil {
						return err
					}

					env := googleSqlUser.CreateUserEnvVars(password)
					vars = google_sql.MapEnvToVars(env, vars)

					secretKeyRefEnvName, err := googleSqlUser.KeyWithSuffixMatchingUser(vars, google_sql.GoogleSQLPasswordSuffix)
					if err != nil {
						return fmt.Errorf("unable to assign sql password: %s", err)
					}

					sqlUser, err := googleSqlUser.Create(app, secretKeyRefEnvName, sqlInstance.CascadingDelete, resourceOptions.GoogleTeamProjectId)
					if err != nil {
						return fmt.Errorf("unable to create sql user: %s", err)
					}
					*operations = append(*operations, resource.Operation{Resource: sqlUser, Operation: resource.OperationCreateIfNotExists})

					scrt := secret.OpaqueSecret(app, google_sql.GoogleSQLSecretName(app, googleSqlUser.Instance.Name, googleSqlUser.Name), vars)
					*operations = append(*operations, resource.Operation{Resource: scrt, Operation: resource.OperationCreateIfNotExists})
				}
			}

			// FIXME: take into account when refactoring default values
			app.Spec.GCP.SqlInstances[i].Name = sqlInstance.Name

			podSpec := &deployment.Spec.Template.Spec
			podSpec = google_sql.AppendGoogleSQLUserSecretEnvs(podSpec, app)
			for _, instance := range app.Spec.GCP.SqlInstances {
				podSpec.Containers = append(podSpec.Containers, google.CloudSqlProxyContainer(instance, 5432, resourceOptions.GoogleTeamProjectId))
			}
			deployment.Spec.Template.Spec = *podSpec
		}
	}
	return nil
}

func createBucket(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, operations *resource.Operations, googleServiceAccount google_iam_crd.IAMServiceAccount) {
	if app.Spec.GCP.Buckets != nil {
		for _, b := range app.Spec.GCP.Buckets {
			bucket := google_storagebucket.GoogleStorageBucket(app, b)
			*operations = append(*operations, resource.Operation{Resource: bucket, Operation: resource.OperationCreateIfNotExists})

			bucketAccessControl := google_storagebucket.GoogleStorageBucketAccessControl(app, bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
			*operations = append(*operations, resource.Operation{Resource: bucketAccessControl, Operation: resource.OperationCreateOrUpdate})

			iamPolicyMember := google_storagebucket.StorageBucketIamPolicyMember(app, bucket, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
			*operations = append(*operations, resource.Operation{Resource: iamPolicyMember, Operation: resource.OperationCreateIfNotExists})
		}
	}
}
