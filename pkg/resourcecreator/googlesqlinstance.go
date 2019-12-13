package resourcecreator

import (
	"fmt"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1alpha3"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AvailabilityTypeRegional = "REGIONAL"
	AvailabilityTypeZonal    = "Zonal"
	GCPRegion                = "europe-north1"
)

func availabilityType(highAvailability bool) string {
	if highAvailability {
		return AvailabilityTypeRegional
	} else {
		return AvailabilityTypeZonal
	}
}

func diskType(diskType nais.CloudSqlInstanceDiskType) string {
	return fmt.Sprintf("PD_%s", diskType)
}

func tier(cpu, memory int) string {
	return fmt.Sprintf("db-custom-%d-%d", cpu, memory)
}

func cascadingDelete(cascadingDelete bool) map[string]string {
	if cascadingDelete {
		return nil
	}

	return map[string]string{"cnrm.cloud.google.com/deletion-policy": "abandon"}
}

func GoogleSqlInstance(app *nais.Application) *google_sql_crd.SqlInstance {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = app.Name
	objectMeta.Annotations = cascadingDelete(app.Spec.GCP.SqlInstance.CascadingDelete)

	return &google_sql_crd.SqlInstance{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SqlInstance",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SqlInstanceSpec{
			DatabaseVersion: string(app.Spec.GCP.SqlInstance.Type),
			Region:          GCPRegion,
			Settings: google_sql_crd.SqlInstanceSettings{
				AvailabilityType:    availabilityType(app.Spec.GCP.SqlInstance.HighAvailability),
				BackupConfiguration: google_sql_crd.SqlInstanceBackupConfiguration{},
				DiskAutoResize:      app.Spec.GCP.SqlInstance.DiskAutoResize,
				DiskSize:            app.Spec.GCP.SqlInstance.DiskSize,
				DiskType:            diskType(app.Spec.GCP.SqlInstance.DiskType),
				Tier:                tier(app.Spec.GCP.SqlInstance.Cpu, app.Spec.GCP.SqlInstance.Memory),
			},
		},
	}
}
