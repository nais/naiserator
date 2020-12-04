package resourcecreator

import (
	"fmt"
	"k8s.io/utils/pointer"

	"github.com/imdario/mergo"

	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func GoogleSqlInstance(app *nais.Application, instance nais.CloudSqlInstance, projectId string) *google_sql_crd.SQLInstance {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = instance.Name
	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)

	if !instance.CascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}

	sqlInstance := &google_sql_crd.SQLInstance{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLInstance",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLInstanceSpec{
			DatabaseVersion: string(instance.Type),
			Region:          GoogleRegion,
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

func CloudSqlInstanceWithDefaults(instance nais.CloudSqlInstance, appName string) (nais.CloudSqlInstance, error) {
	var err error

	defaultInstance := nais.CloudSqlInstance{
		Name:      appName,
		Tier:      DefaultSqlInstanceTier,
		DiskType:  DefaultSqlInstanceDiskType,
		DiskSize:  DefaultSqlInstanceDiskSize,
		Databases: []nais.CloudSqlDatabase{{Name: appName}},
		Collation: DefaultSqlInstanceCollation,
	}

	if err = mergo.Merge(&instance, defaultInstance); err != nil {
		return nais.CloudSqlInstance{}, fmt.Errorf("unable to merge default sqlinstance values: %s", err)
	}

	// Have to do this check explicitly as mergo is not able to distingush between nil pointer and 0.
	if instance.AutoBackupHour == nil {
		instance.AutoBackupHour = intp(DefaultSqlInstanceAutoBackupHour)
	}

	return instance, err
}

func availabilityType(highAvailability bool) string {
	if highAvailability {
		return AvailabilityTypeRegional
	} else {
		return AvailabilityTypeZonal
	}
}

func SqlInstanceIamPolicyMember(app *nais.Application, resourceName string, googleProjectId, googleTeamProjectId string) *google_iam_crd.IAMPolicyMember {
	policy := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: (*app).CreateObjectMetaWithName(resourceName),
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: GoogleIAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", GcpServiceAccountName(app, googleProjectId)),
			Role:   "roles/cloudsql.client",
			ResourceRef: google_iam_crd.ResourceRef{
				Kind: "Project",
				Name: pointer.StringPtr(""),
			},
		},
	}

	setAnnotation(policy, GoogleProjectIdAnnotation, googleTeamProjectId)

	return policy
}
