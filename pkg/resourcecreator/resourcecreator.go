// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/keygen"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/idporten"
	"github.com/nais/naiserator/pkg/resourcecreator/ingress"
	"github.com/nais/naiserator/pkg/resourcecreator/kafka"
	"github.com/nais/naiserator/pkg/resourcecreator/maskinporten"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resourceutils"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
)

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais_io_v1alpha1.Application, resourceOptions resourceutils.Options) (ResourceOperations, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ops := ResourceOperations{
		{service.Service(app), OperationCreateOrUpdate},
		{serviceaccount.ServiceAccount(app, resourceOptions), OperationCreateIfNotExists},
		{horizontalpodautoscaler.HorizontalPodAutoscaler(app), OperationCreateOrUpdate},
	}

	outboundHostRules := make([]nais_io_v1.AccessPolicyExternalRule, 0)

	pdb := poddisruptionbudget.PodDisruptionBudget(app)
	if pdb != nil {
		ops = append(ops, ResourceOperation{pdb, OperationCreateOrUpdate})
	}

	if app.Spec.LeaderElection {
		leRole := LeaderElectionRole(app)
		leRoleBinding := LeaderElectionRoleBinding(app)
		ops = append(ops, ResourceOperation{leRole, OperationCreateOrUpdate})
		ops = append(ops, ResourceOperation{leRoleBinding, OperationCreateOrRecreate})
	}

	if resourceOptions.JwkerEnabled && app.Spec.TokenX.Enabled {
		jwker := Jwker(app, resourceOptions.ClusterName)
		if jwker != nil {
			outboundHostRules = append(outboundHostRules, ToAccessPolicyExternalRules(resourceOptions.JwkerHosts)...)

			ops = append(ops, ResourceOperation{jwker, OperationCreateOrUpdate})
			resourceOptions.JwkerSecretName = jwker.Spec.SecretName
		}
	}

	if resourceOptions.AzureratorEnabled && app.Spec.Azure.Application.Enabled {
		azureapp, err := AzureAdApplication(*app, resourceOptions.ClusterName)
		if err != nil {
			return nil, err
		}
		outboundHostRules = append(outboundHostRules, ToAccessPolicyExternalRules(resourceOptions.AzureratorHosts)...)

		ops = append(ops, ResourceOperation{&azureapp, OperationCreateOrUpdate})
		resourceOptions.AzureratorSecretName = azureapp.Spec.SecretName
	}

	if resourceOptions.KafkaratorEnabled && app.Spec.Kafka != nil {
		var err error
		resourceOptions.KafkaratorSecretName, err = kafka.GenerateKafkaSecretName(app)
		if err != nil {
			return nil, err
		}
	}

	if resourceOptions.DigdiratorEnabled && app.Spec.IDPorten != nil && app.Spec.IDPorten.Enabled {
		idportenClient, err := idporten.IDPortenClient(app)
		if err != nil {
			return nil, err
		}
		outboundHostRules = append(outboundHostRules, ToAccessPolicyExternalRules(resourceOptions.DigdiratorHosts)...)

		ops = append(ops, ResourceOperation{idportenClient, OperationCreateOrUpdate})
		resourceOptions.DigdiratorIDPortenSecretName = idportenClient.Spec.SecretName
	}

	if resourceOptions.DigdiratorEnabled && app.Spec.Maskinporten != nil && app.Spec.Maskinporten.Enabled {
		maskinportenClient := maskinporten.MaskinportenClient(app)

		outboundHostRules = append(outboundHostRules, ToAccessPolicyExternalRules(resourceOptions.DigdiratorHosts)...)

		ops = append(ops, ResourceOperation{maskinportenClient, OperationCreateOrUpdate})
		resourceOptions.DigdiratorMaskinportenSecretName = maskinportenClient.Spec.SecretName
	}

	if app.Spec.Elastic != nil {
		env := strings.Split(resourceOptions.ClusterName, "-")[0]
		instanceName := fmt.Sprintf("elastic-%s-%s-nav-%s.aivencloud.com", team, app.Spec.Elastic.Instance, env)
		outboundHostRules = append(outboundHostRules, nais_io_v1.AccessPolicyExternalRule{
			Host: instanceName,
			Ports: []nais_io_v1.AccessPolicyPortRule{
				{
					Name:     "https",
					Port:     26482,
					Protocol: "HTTPS",
				},
			},
		})
	}

	if len(resourceOptions.GoogleProjectId) > 0 {
		googleServiceAccount := google_iam.GoogleIAMServiceAccount(app, resourceOptions.GoogleProjectId)
		googleServiceAccountBinding := google_iam.GoogleIAMPolicy(app, &googleServiceAccount, resourceOptions.GoogleProjectId)
		ops = append(ops, ResourceOperation{&googleServiceAccount, OperationCreateOrUpdate})
		ops = append(ops, ResourceOperation{&googleServiceAccountBinding, OperationCreateOrUpdate})

		if app.Spec.GCP != nil && app.Spec.GCP.Buckets != nil && len(app.Spec.GCP.Buckets) > 0 {
			for _, b := range app.Spec.GCP.Buckets {
				bucket := google_storagebucket.GoogleStorageBucket(app, b)
				ops = append(ops, ResourceOperation{bucket, OperationCreateIfNotExists})

				bucketAccessControl := google_storagebucket.GoogleStorageBucketAccessControl(app, bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
				ops = append(ops, ResourceOperation{bucketAccessControl, OperationCreateOrUpdate})

				iamPolicyMember := google_storagebucket.StorageBucketIamPolicyMember(app, bucket, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, ResourceOperation{iamPolicyMember, OperationCreateIfNotExists})
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
				ops = append(ops, ResourceOperation{instance, OperationCreateOrUpdate})

				iamPolicyMember := google_sql.SqlInstanceIamPolicyMember(app, sqlInstance.Name, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
				ops = append(ops, ResourceOperation{iamPolicyMember, OperationCreateIfNotExists})

				for _, db := range sqlInstance.Databases {
					sqlUsers := google_sql.MergeAndFilterSQLUsers(db.Users, instance.Name)

					googledb := google_sql.GoogleSQLDatabase(app, db, sqlInstance, resourceOptions.GoogleTeamProjectId)
					ops = append(ops, ResourceOperation{googledb, OperationCreateIfNotExists})

					for _, user := range sqlUsers {
						vars := make(map[string]string)

						googleSqlUser := google_sql.SetupNewGoogleSqlUser(user.Name, &db, instance)

						password, err := generatePassword()
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
						ops = append(ops, ResourceOperation{sqlUser, OperationCreateIfNotExists})

						scrt := secret.OpaqueSecret(app, google_sql.GoogleSQLSecretName(app, googleSqlUser.Instance.Name, googleSqlUser.Name), vars)
						ops = append(ops, ResourceOperation{scrt, OperationCreateIfNotExists})
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
				ops = append(ops, ResourceOperation{policy, OperationCreateIfNotExists})
			}
		}
	}

	if resourceOptions.NetworkPolicy {
		ops = append(ops, ResourceOperation{NetworkPolicy(app, resourceOptions), OperationCreateOrUpdate})
	}

	if !resourceOptions.Linkerd {
		ing, err := ingress.Ingress(app)
		if err != nil {
			return nil, fmt.Errorf("while creating ingress: %s", err)
		}

		if ing != nil {
			ops = append(ops, ResourceOperation{ing, OperationCreateOrUpdate})
		}
	}

	if resourceOptions.Linkerd {
		ingresses, err := ingress.NginxIngresses(app, resourceOptions)
		if err != nil {
			return nil, fmt.Errorf("while creating ingresses: %s", err)
		}

		if ingresses != nil {
			for _, ing := range ingresses {
				ops = append(ops, ResourceOperation{ing, OperationCreateOrUpdate})
			}
		}
	}

	deployment, err := Deployment(app, resourceOptions)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	ops = append(ops, ResourceOperation{deployment, OperationCreateOrUpdate})

	return ops, nil
}

func generatePassword() (string, error) {
	key, err := keygen.Keygen(32)
	if err != nil {
		return "", fmt.Errorf("unable to generate secret for sql user: %s", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(key), nil
}
