package google_sql

import (
	"fmt"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/pointer"

	"github.com/imdario/mergo"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AvailabilityTypeRegional         = "REGIONAL"
	AvailabilityTypeZonal            = "ZONAL"
	DefaultSqlInstanceDiskType       = nais_io_v1.CloudSqlInstanceDiskTypeSSD
	DefaultSqlInstanceAutoBackupHour = 2
	DefaultSqlInstanceTier           = "db-f1-micro"
	DefaultSqlInstanceDiskSize       = 10
	DefaultSqlInstanceCollation      = "en_US.UTF8"
)

type Source interface {
	resource.Source
	GetGCP() *nais_io_v1.GCP
}

type Config interface {
	GetGoogleProjectID() string
	GetGoogleTeamProjectID() string
	GetGoogleCloudSQLProxyContainerImage() string
	ShouldCreateSqlInstanceInSharedVpc() bool
}

func availabilityType(highAvailability bool) string {
	if highAvailability {
		return AvailabilityTypeRegional
	} else {
		return AvailabilityTypeZonal
	}
}

func GoogleSqlInstance(objectMeta metav1.ObjectMeta, instance nais_io_v1.CloudSqlInstance, cfg Config) *google_sql_crd.SQLInstance {
	objectMeta.Name = instance.Name
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, cfg.GetGoogleTeamProjectID())
	util.SetAnnotation(&objectMeta, google.StateIntoSpec, google.StateIntoSpecValue)

	if !instance.CascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
	}

	var backupSettings *google_sql_crd.SQLInstanceBackupRetentionSetting
	if instance.RetainedBackups != nil {
		backupSettings = &google_sql_crd.SQLInstanceBackupRetentionSetting{
			RetainedBackups: *instance.RetainedBackups,
		}
	}

	flags := []google_sql_crd.SQLDatabaseFlag{{Name: "cloudsql.iam_authentication", Value: "on"}}
	for _, flag := range instance.Flags {
		err := ValidateFlag(flag.Name, flag.Value)
		if err != nil {
			log.Errorf("sql instance flag '%s' is not valid: %v", flag.Name, err)
		}
		flags = append(flags, google_sql_crd.SQLDatabaseFlag{Name: flag.Name, Value: flag.Value})
	}

	var privateNetworkRef *google_sql_crd.PrivateNetworkRef
	if cfg != nil && cfg.ShouldCreateSqlInstanceInSharedVpc() {
		privateNetworkRef = &google_sql_crd.PrivateNetworkRef{
			External: "projects/" + cfg.GetGoogleProjectID() + "/global/networks/nais-vpc",
		}
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
					Enabled:                    true,
					StartTime:                  fmt.Sprintf("%02d:00", *instance.AutoBackupHour),
					PointInTimeRecoveryEnabled: instance.PointInTimeRecovery,
					BackupRetentionSettings:    backupSettings,
				},
				IpConfiguration: google_sql_crd.SQLInstanceIpConfiguration{
					RequireSsl:        true,
					PrivateNetworkRef: privateNetworkRef,
				},
				DiskAutoresize: instance.DiskAutoresize,
				DiskSize:       instance.DiskSize,
				DiskType:       instance.DiskType.GoogleType(),
				Tier:           instance.Tier,
				DatabaseFlags:  flags,
				InsightsConfig: google_sql_crd.SQLInstanceInsightsConfiguration{
					QueryInsightsEnabled: instance.Insights.IsEnabled(),
				},
			},
		},
	}

	if instance.Insights != nil {
		sqlInstance.Spec.Settings.InsightsConfig.QueryStringLength = instance.Insights.QueryStringLength
		sqlInstance.Spec.Settings.InsightsConfig.RecordApplicationTags = instance.Insights.RecordApplicationTags
		sqlInstance.Spec.Settings.InsightsConfig.RecordClientAddress = instance.Insights.RecordClientAddress
	}

	if instance.Maintenance != nil && instance.Maintenance.Hour != nil && instance.Maintenance.Day != 0 {
		sqlInstance.Spec.Settings.MaintenanceWindow = &google_sql_crd.MaintenanceWindow{
			Day:  instance.Maintenance.Day,
			Hour: *instance.Maintenance.Hour,
		}
	}

	return sqlInstance
}

func CloudSqlInstanceWithDefaults(instance nais_io_v1.CloudSqlInstance, appName string) (nais_io_v1.CloudSqlInstance, error) {
	defaultInstance := nais_io_v1.CloudSqlInstance{
		Tier:     DefaultSqlInstanceTier,
		DiskType: DefaultSqlInstanceDiskType,
		DiskSize: DefaultSqlInstanceDiskSize,
		// This default will always be overridden by GoogleSQLDatabase(), need to be set, as databases.Name can not be nil.
		Databases: []nais_io_v1.CloudSqlDatabase{{Name: "dummy-name"}},
		Collation: DefaultSqlInstanceCollation,
	}

	err := mergo.Merge(&instance, defaultInstance)
	if err != nil {
		return nais_io_v1.CloudSqlInstance{}, fmt.Errorf("unable to merge default sqlinstance values: %s", err)
	}

	// Have to do this check explicitly as mergo is not able to distinguish between nil pointer and 0.
	if instance.AutoBackupHour == nil {
		instance.AutoBackupHour = util.Intp(DefaultSqlInstanceAutoBackupHour)
	}

	if instance.Name == "" {
		instance.Name = appName
	}

	return instance, nil
}

func instanceIamPolicyMember(source resource.Source, resourceName string, cfg Config) *google_iam_crd.IAMPolicyMember {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = resourceName
	policy := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf(
				"serviceAccount:%s",
				google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), cfg.GetGoogleProjectID()),
			),
			Role: "roles/cloudsql.client",
			ResourceRef: google_iam_crd.ResourceRef{
				Kind: "Project",
				Name: pointer.StringPtr(""),
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, cfg.GetGoogleTeamProjectID())

	return policy
}

func createSqlUserDBResources(objectMeta metav1.ObjectMeta, ast *resource.Ast, googleSqlUser GoogleSqlUser, cascadingDelete bool, appName, googleTeamProjectId string) error {
	vars := make(map[string]string)

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

	secretName, err := GoogleSQLSecretName(
		appName, googleSqlUser.Instance.Name, googleSqlUser.DB.Name, googleSqlUser.Name,
	)
	if err != nil {
		return fmt.Errorf("unable to create sql secret name: %s", err)
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, secret.OpaqueSecret(objectMeta, secretName, vars))
	ast.AppendOperation(resource.AnnotateIfExists, secret.OpaqueSecret(objectMeta, secretName, nil))

	sqlUser, err := googleSqlUser.Create(objectMeta, cascadingDelete, secretKeyRefEnvName, appName, googleTeamProjectId)
	if err != nil {
		return fmt.Errorf("unable to create sql user: %s", err)
	}
	ast.AppendOperation(resource.OperationCreateIfNotExists, sqlUser)
	return nil
}

func CreateInstance(source Source, ast *resource.Ast, cfg Config) error {
	gcp := source.GetGCP()
	if gcp == nil {
		return nil
	}

	sourceName := source.GetName()

	for i, sqlInstance := range gcp.SqlInstances {
		// This could potentially be removed to add possibility for several instances.
		if i > 0 {
			return fmt.Errorf("only one sql instance is supported")
		}

		sqlInstance, err := CloudSqlInstanceWithDefaults(sqlInstance, sourceName)
		if err != nil {
			return err
		}

		objectMeta := resource.CreateObjectMeta(source)
		googleTeamProjectId := cfg.GetGoogleTeamProjectID()

		googleSqlInstance := GoogleSqlInstance(objectMeta, sqlInstance, cfg)
		ast.AppendOperation(resource.OperationCreateOrUpdate, googleSqlInstance)

		iamPolicyMember := instanceIamPolicyMember(source, googleSqlInstance.Name, cfg)
		ast.AppendOperation(resource.OperationCreateIfNotExists, iamPolicyMember)

		for dbNum, db := range sqlInstance.Databases {

			googledb := GoogleSQLDatabase(
				objectMeta, googleSqlInstance.Name, db.Name, googleTeamProjectId, sqlInstance.CascadingDelete,
			)
			ast.AppendOperation(resource.OperationCreateIfNotExists, googledb)

			sqlUsers, err := MergeAndFilterDatabaseSQLUsers(db.Users, googleSqlInstance.Name, dbNum)
			if err != nil {
				return err
			}

			for _, user := range sqlUsers {
				googleSqlUser := SetupGoogleSqlUser(user.Name, &db, googleSqlInstance)
				if err = createSqlUserDBResources(
					objectMeta, ast, googleSqlUser, sqlInstance.CascadingDelete, sourceName, googleTeamProjectId,
				); err != nil {
					return err
				}
			}
		}

		if defaultNameNotSetInManifest(gcp.SqlInstances[i]) {
			gcp.SqlInstances[i].Name = sqlInstance.Name
		}

		err = AppendGoogleSQLUserSecretEnvs(ast, sqlInstance, sourceName)
		if err != nil {
			return fmt.Errorf("unable to append sql user secret envs: %s", err)
		}
		ast.Containers = append(
			ast.Containers, google.CloudSqlProxyContainer(5432, cfg.GetGoogleCloudSQLProxyContainerImage(), cfg.GetGoogleTeamProjectID(), googleSqlInstance.Name),
		)
	}
	return nil
}

func defaultNameNotSetInManifest(instance nais_io_v1.CloudSqlInstance) bool {
	return instance.Name == ""
}
