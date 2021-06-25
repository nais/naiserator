package google_sql

import (
	"fmt"
	"strconv"

	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/pointer"

	"github.com/imdario/mergo"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AvailabilityTypeRegional         = "REGIONAL"
	AvailabilityTypeZonal            = "ZONAL"
	DefaultSqlInstanceDiskType       = nais.CloudSqlInstanceDiskTypeSSD
	DefaultSqlInstanceAutoBackupHour = 2
	DefaultSqlInstanceTier           = "db-f1-micro"
	DefaultSqlInstanceDiskSize       = 10
	DefaultSqlInstanceCollation      = "en_US.UTF8"
)

func GoogleSqlInstance(objectMeta metav1.ObjectMeta, instance nais.CloudSqlInstance, projectId string) *google_sql_crd.SQLInstance {
	objectMeta.Name = instance.Name
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)

	if !instance.CascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
	}

	sqlInstance := &google_sql_crd.SQLInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SQLInstance",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLInstanceSpec{
			DatabaseVersion: string(instance.Type),
			Region:          google.Region,
			Settings: google_sql_crd.SQLInstanceSettings{
				AvailabilityType: availabilityType(instance.HighAvailability),
				BackupConfiguration: google_sql_crd.SQLInstanceBackupConfiguration{
					Enabled:   true,
					StartTime: fmt.Sprintf("%02d:00", *instance.AutoBackupHour),
				},
				IpConfiguration: google_sql_crd.SQLInstanceIpConfiguration{
					RequireSsl: true,
				},
				DiskAutoresize: instance.DiskAutoresize,
				DiskSize:       instance.DiskSize,
				DiskType:       instance.DiskType.GoogleType(),
				Tier:           instance.Tier,
				DatabaseFlags:  []google_sql_crd.SQLDatabaseFlag{{Name: "cloudsql.iam_authentication", Value: "on"}},
			},
		},
	}

	if instance.Maintenance != nil && instance.Maintenance.Hour != nil && instance.Maintenance.Day != 0 {
		sqlInstance.Spec.Settings.MaintenanceWindow = &google_sql_crd.MaintenanceWindow{
			Day:  instance.Maintenance.Day,
			Hour: *instance.Maintenance.Hour,
		}
	}

	return sqlInstance
}

func CloudSqlInstanceWithDefaults(instance nais.CloudSqlInstance, appName string, instanceNumber int) (nais.CloudSqlInstance, error) {
	var err error

	instanceName, err := UniqueName(appName, "instance", instanceNumber)
	dbName, err := UniqueName(appName, "db", instanceNumber)

	defaultInstance := nais.CloudSqlInstance{
		Tier:     DefaultSqlInstanceTier,
		DiskType: DefaultSqlInstanceDiskType,
		DiskSize: DefaultSqlInstanceDiskSize,
		// This default name will be overridden by CreateDatabase
		Databases: []nais.CloudSqlDatabase{{Name: dbName}},
		Collation: DefaultSqlInstanceCollation,
	}

	if err = mergo.Merge(&instance, defaultInstance); err != nil {
		return nais.CloudSqlInstance{}, fmt.Errorf("unable to merge default sqlinstance values: %s", err)
	}

	// Have to do this check explicitly as mergo is not able to distinguish between nil pointer and 0.
	if instance.AutoBackupHour == nil {
		instance.AutoBackupHour = util.Intp(DefaultSqlInstanceAutoBackupHour)
	}

	instance.Name = instanceName

	return instance, err
}

func UniqueName(basename, resourceType string, resourceNumber int) (string, error) {
	number := strconv.Itoa(resourceNumber)
	if number != "0" {
		addedSuffix := fmt.Sprintf("%s-%s", resourceType, number)
		return namegen.ShortName(fmt.Sprintf("%s-%s", basename, addedSuffix), validation.DNS1035LabelMaxLength)
	}
	return basename, nil
}

func availabilityType(highAvailability bool) string {
	if highAvailability {
		return AvailabilityTypeRegional
	} else {
		return AvailabilityTypeZonal
	}
}

func instanceIamPolicyMember(source resource.Source, resourceName, googleProjectId, googleTeamProjectId string) *google_iam_crd.IAMPolicyMember {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = resourceName
	policy := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), googleProjectId)),
			Role:   "roles/cloudsql.client",
			ResourceRef: google_iam_crd.ResourceRef{
				Kind: "Project",
				Name: pointer.StringPtr(""),
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, googleTeamProjectId)

	return policy
}

func CreateInstance(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisSqlInstances *[]nais.CloudSqlInstance) error {
	if naisSqlInstances == nil {
		return nil
	}

	for i, sqlInstance := range *naisSqlInstances {
		if i > 0 {
			return fmt.Errorf("only one sql instance is supported")
		}

		sqlInstance, err := CloudSqlInstanceWithDefaults(sqlInstance, source.GetName(), i)
		if err != nil {
			return err
		}

		objectMeta := resource.CreateObjectMeta(source)

		instance := GoogleSqlInstance(objectMeta, sqlInstance, resourceOptions.GoogleTeamProjectId)
		ast.AppendOperation(resource.OperationCreateOrUpdate, instance)

		iamPolicyMember := instanceIamPolicyMember(source, sqlInstance.Name, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
		ast.AppendOperation(resource.OperationCreateIfNotExists, iamPolicyMember)

		for _, db := range sqlInstance.Databases {
			sqlUsers := MergeAndFilterSQLUsers(db.Users, instance.Name)

			googledb := GoogleSQLDatabase(objectMeta, db, sqlInstance, resourceOptions.GoogleTeamProjectId)
			ast.AppendOperation(resource.OperationCreateIfNotExists, googledb)

			for _, user := range sqlUsers {
				vars := make(map[string]string)

				googleSqlUser := SetupNewGoogleSqlUser(user.Name, &db, instance)

				password, err := util.GeneratePassword()
				if err != nil {
					return err
				}

				env := googleSqlUser.CreateUserEnvVars(password)
				vars = MapEnvToVars(env, vars)

				secretKeyRefEnvName, err := googleSqlUser.KeyWithSuffixMatchingUser(vars, GoogleSQLPasswordSuffix)
				if err != nil {
					return fmt.Errorf("unable to assign sql password: %s", err)
				}

				secretName, err := GoogleSQLSecretName(source.GetName(), googleSqlUser.Instance.Name, db.Name, googleSqlUser.Name)
				if err != nil {
					return fmt.Errorf("unable to create sql secret name: %s", err)
				}

				scrt := secret.OpaqueSecret(objectMeta, secretName, vars)
				ast.AppendOperation(resource.OperationCreateIfNotExists, scrt)

				sqlUser, err := googleSqlUser.Create(objectMeta, secretKeyRefEnvName, db.Name, sqlInstance.CascadingDelete, resourceOptions.GoogleTeamProjectId)
				if err != nil {
					return fmt.Errorf("unable to create sql user: %s", err)
				}
				ast.AppendOperation(resource.OperationCreateIfNotExists, sqlUser)
			}
		}

		if err = AppendGoogleSQLUserSecretEnvs(ast, naisSqlInstances, source.GetName()); err != nil {
			return fmt.Errorf("unable to append sql user secret envs: %s", err)
		}

		(*naisSqlInstances)[i].Name = sqlInstance.Name
		for _, instance := range *naisSqlInstances {
			ast.Containers = append(ast.Containers, google.CloudSqlProxyContainer(5432, resourceOptions.GoogleCloudSQLProxyContainerImage, resourceOptions.GoogleTeamProjectId, instance.Name))
		}
	}
	return nil
}
