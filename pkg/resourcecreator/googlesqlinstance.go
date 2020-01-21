package resourcecreator

import (
	"fmt"

	"github.com/imdario/mergo"

	google_iam_crd "github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AvailabilityTypeRegional     = "REGIONAL"
	AvailabilityTypeZonal        = "ZONAL"
	DefaultSqlInstanceDiskType   = nais.CloudSqlInstanceDiskTypeSSD
	DefaultSqlInstanceAutoBackup = "02:00"
	DefaultSqlInstanceTier       = "db-f1-micro"
	DefaultSqlInstanceDiskSize   = 10
)

func GoogleSqlInstance(app *nais.Application, instance nais.CloudSqlInstance, projectId string) *google_sql_crd.SQLInstance {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = instance.Name
	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)

	if !instance.CascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}

	return &google_sql_crd.SQLInstance{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLInstance",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLInstanceSpec{
			DatabaseVersion: string(instance.Type),
			Region:          GoogleRegion,
			Settings: google_sql_crd.SQLInstanceSettings{
				AvailabilityType:    availabilityType(instance.HighAvailability),
				BackupConfiguration: google_sql_crd.SQLInstanceBackupConfiguration{},
				DiskAutoresize:      instance.DiskAutoresize,
				DiskSize:            instance.DiskSize,
				DiskType:            instance.DiskType.GoogleType(),
				Tier:                instance.Tier,
			},
		},
	}
}

func CloudSqlInstanceWithDefaults(instance nais.CloudSqlInstance, appName string) (nais.CloudSqlInstance, error) {
	var err error

	defaultInstance := nais.CloudSqlInstance{
		Name:           appName,
		Tier:           DefaultSqlInstanceTier,
		DiskType:       DefaultSqlInstanceDiskType,
		DiskSize:       DefaultSqlInstanceDiskSize,
		AutoBackupTime: DefaultSqlInstanceAutoBackup,
		Databases:      []nais.CloudSqlDatabase{{Name: appName}},
	}

	if err = mergo.Merge(&instance, defaultInstance); err != nil {
		return nais.CloudSqlInstance{}, fmt.Errorf("unable to merge default sqlinstance values: %s", err)
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

func SqlInstanceIamPolicyMember(app *nais.Application, resourceName string, googleProjectId string) *google_iam_crd.IAMPolicyMember {
	return &google_iam_crd.IAMPolicyMember{
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
			},
		},
	}
}
