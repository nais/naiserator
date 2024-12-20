package google_sql

import (
	"fmt"

	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
)

const (
	AvailabilityTypeRegional         = "REGIONAL"
	AvailabilityTypeZonal            = "ZONAL"
	DefaultSqlInstanceAutoBackupHour = 2
	DefaultSqlInstanceCollation      = "en_US.UTF8"
	DefaultSqlInstanceDiskSize       = 10
	DefaultSqlInstanceDiskType       = nais_io_v1.CloudSqlInstanceDiskTypeSSD
	DefaultSqlInstanceEdition        = google_sql_crd.SQLInstanceEditionEnterprise

	sqeletorVolumeName = "sqeletor-sql-ssl-cert"

	maxLengthName = 63
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
	SqlInstanceExists() bool
	SqlInstanceHasPrivateIpInSharedVpc() bool
	GetClusterName() string
}

func CreateInstance(source Source, ast *resource.Ast, cfg Config) error {
	manifestGCP := source.GetGCP()
	if manifestGCP == nil {
		return nil
	}

	sourceName := source.GetName()

	// Short-circuit into NOOP
	if len(manifestGCP.SqlInstances) == 0 {
		return nil
	}

	if len(manifestGCP.SqlInstances) > 1 {
		return fmt.Errorf("only one sql instance is supported even though the spec indicates otherwise")
	}

	naisSqlInstance, err := NaisCloudSqlInstanceWithDefaults(manifestGCP.Instance(), sourceName)
	if err != nil {
		return err
	}

	if len(naisSqlInstance.Databases) > 1 {
		return fmt.Errorf("only one sql database is supported even though the spec indicates otherwise")
	}

	naisSqlDatabase := naisSqlInstance.Database()
	googleTeamProjectID := cfg.GetGoogleTeamProjectID()

	googleSqlInstance, err := CreateGoogleSqlInstance(resource.CreateObjectMeta(source), naisSqlInstance, cfg)
	if err != nil {
		return err
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, googleSqlInstance)

	googleIAMPolicyMember := CreateIAMPolicyMemberForInstance(source, googleSqlInstance.Name, cfg)
	ast.AppendOperation(resource.OperationCreateIfNotExists, googleIAMPolicyMember)

	googleSqlDatabase := CreateGoogleSQLDatabase(resource.CreateObjectMeta(source), googleSqlInstance.Name, naisSqlDatabase.Name, googleTeamProjectID, naisSqlInstance.CascadingDelete)
	ast.AppendOperation(resource.OperationCreateOrUpdate, googleSqlDatabase)

	CreateGoogleSQLUsers(source, ast, cfg, naisSqlDatabase, naisSqlInstance, googleSqlInstance)

	needsCloudSqlProxyContainer := true
	if cfg != nil && cfg.ShouldCreateSqlInstanceInSharedVpc() {
		if usingPrivateIP(googleSqlInstance) {
			needsCloudSqlProxyContainer = false
			CreateSqlSSLCertResource(ast, googleSqlInstance.Name, source, googleTeamProjectID, cfg.GetClusterName())
		}
	}

	if needsCloudSqlProxyContainer {
		cloudSqlProxyContainer := google.CloudSqlProxyContainer(5432, cfg.GetGoogleCloudSQLProxyContainerImage(), googleTeamProjectID, googleSqlInstance.Name)
		ast.Containers = append(ast.Containers, cloudSqlProxyContainer)
	}

	return nil
}

func availabilityType(highAvailability bool) string {
	if highAvailability {
		return AvailabilityTypeRegional
	} else {
		return AvailabilityTypeZonal
	}
}

func CreateGoogleSqlInstance(objectMeta metav1.ObjectMeta, instance *nais_io_v1.CloudSqlInstance, cfg Config) (*google_sql_crd.SQLInstance, error) {
	objectMeta.Name = instance.Name
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, cfg.GetGoogleTeamProjectID())

	if len(instance.Tier) == 0 {
		return nil, fmt.Errorf("DB instance tier missing. Previous default value was `db-f1-micro` (recommended only for development); closest recommended value for production use is `db-custom-1-3840`")
	}

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

	if instance.TransactionLogRetentionDays != nil {
		if backupSettings == nil {
			backupSettings = &google_sql_crd.SQLInstanceBackupRetentionSetting{}
		}
		backupSettings.TransactionLogRetentionDays = *instance.TransactionLogRetentionDays
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
		if cfg.SqlInstanceHasPrivateIpInSharedVpc() || !cfg.SqlInstanceExists() {
			privateNetworkRef = &google_sql_crd.PrivateNetworkRef{
				External: "projects/" + cfg.GetGoogleProjectID() + "/global/networks/nais-vpc",
			}
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
				Edition: DefaultSqlInstanceEdition,
				IpConfiguration: google_sql_crd.SQLInstanceIpConfiguration{
					RequireSsl:        true,
					PrivateNetworkRef: privateNetworkRef,
				},
				DiskAutoresize:      instance.DiskAutoresize,
				DiskAutoresizeLimit: instance.DiskAutoresizeLimit,
				DiskSize:            instance.DiskSize,
				DiskType:            instance.DiskType.GoogleType(),
				Tier:                instance.Tier,
				DatabaseFlags:       flags,
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

	return sqlInstance, nil
}

func NaisCloudSqlInstanceWithDefaults(instance *nais_io_v1.CloudSqlInstance, appName string) (*nais_io_v1.CloudSqlInstance, error) {
	if instance == nil {
		return nil, fmt.Errorf("sql instance not defined")
	}

	defaultInstance := nais_io_v1.CloudSqlInstance{
		DiskType: DefaultSqlInstanceDiskType,
		DiskSize: DefaultSqlInstanceDiskSize,
		// This default will be further formatted by CreateGoogleSQLDatabase().
		Databases: []nais_io_v1.CloudSqlDatabase{{Name: appName}},
		Collation: DefaultSqlInstanceCollation,
	}

	err := mergo.Merge(instance, defaultInstance)
	if err != nil {
		return nil, fmt.Errorf("unable to merge default sqlinstance values: %s", err)
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

func CreateIAMPolicyMemberForInstance(source resource.Source, resourceName string, cfg Config) *google_iam_crd.IAMPolicyMember {
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
				Name: ptr.To(""),
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, cfg.GetGoogleTeamProjectID())

	return policy
}

func usingPrivateIP(googleSqlInstance *google_sql_crd.SQLInstance) bool {
	return googleSqlInstance.Spec.Settings.IpConfiguration.PrivateNetworkRef != nil
}

func CreateSqlSSLCertResource(ast *resource.Ast, instanceName string, source Source, googleTeamProjectId string, clusterName string) {
	objectMeta := resource.CreateObjectMeta(source)
	shortName, err := namegen.ShortName(fmt.Sprintf("%s-%s", source.GetName(), instanceName), maxLengthName)
	if err != nil {
		panic(err) // Will never happen
	}
	objectMeta.Name = shortName

	secretName, err := namegen.ShortName(fmt.Sprintf("sqeletor-%s", instanceName), maxLengthName)
	if err != nil {
		panic(err) // Will never happen
	}

	commonName := namegen.RandShortName(fmt.Sprintf("%s.%s", source.GetName(), clusterName), maxLengthName)

	sqlSSLCert := &google_sql_crd.SQLSSLCert{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SQLSSLCert",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLSSLCertSpec{
			CommonName: commonName,
			InstanceRef: google_sql_crd.InstanceRef{
				Name:      instanceName,
				Namespace: source.GetNamespace(),
			},
		},
	}

	util.SetAnnotation(sqlSSLCert, google.ProjectIdAnnotation, googleTeamProjectId)
	util.SetAnnotation(sqlSSLCert, "sqeletor.nais.io/secret-name", secretName)

	ast.Volumes = append(ast.Volumes, pod.FromFilesSecretVolumeWithMode(sqeletorVolumeName, secretName, nil, ptr.To(int32(0o640))))
	ast.VolumeMounts = append(ast.VolumeMounts, pod.FromFilesVolumeMount(sqeletorVolumeName, nais_io_v1alpha1.DefaultSqeletorMountPath, "", true))

	ast.AppendOperation(resource.OperationCreateIfNotExists, sqlSSLCert)
}