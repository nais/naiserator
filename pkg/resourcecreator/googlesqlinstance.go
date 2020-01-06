package resourcecreator

import (
	"fmt"

	"github.com/imdario/mergo"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/naiserator/pkg/apis/sql.cnrm.cloud.google.com/v1alpha3"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AvailabilityTypeRegional     = "REGIONAL"
	AvailabilityTypeZonal        = "Zonal"
	GCPRegion                    = "europe-north1"
	DefaultSqlInstanceDiskType   = "SSD"
	DefaultSqlInstanceAutoBackup = "02:00"
	DefaultSqlInstanceTier       = "db-f1-micro"
	DefaultSqlInstanceDiskSize   = 10
)

func GoogleSqlInstance(app *nais.Application, instance nais.CloudSqlInstance) (*google_sql_crd.SQLInstance, error) {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = instance.Name

	i, err := withDefaults(instance, app.Name)
	if err != nil {
		return nil, err
	}

	objectMeta.Annotations = CascadingDeleteAnnotation(i.CascadingDelete)

	return &google_sql_crd.SQLInstance{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "SQLInstance",
			APIVersion: "sql.cnrm.cloud.google.com/v1alpha3",
		},
		ObjectMeta: objectMeta,
		Spec: google_sql_crd.SQLInstanceSpec{
			DatabaseVersion: string(i.Type),
			Region:          GCPRegion,
			Settings: google_sql_crd.SQLInstanceSettings{
				AvailabilityType:    availabilityType(i.HighAvailability),
				BackupConfiguration: google_sql_crd.SQLInstanceBackupConfiguration{},
				DiskAutoResize:      i.DiskAutoResize,
				DiskSize:            i.DiskSize,
				DiskType:            diskType(i.DiskType),
				Tier:                i.Tier,
			},
		},
	}, nil
}

func withDefaults(instance nais.CloudSqlInstance, appName string) (withDefaults nais.CloudSqlInstance, err error) {
	defaultInstance := nais.CloudSqlInstance{
		Tier:       DefaultSqlInstanceTier,
		DiskType:   DefaultSqlInstanceDiskType,
		DiskSize:   DefaultSqlInstanceDiskSize,
		AutoBackup: DefaultSqlInstanceAutoBackup,
		Databases:  []nais.CloudSqlDatabase{{Name: appName}},
	}

	if err := mergo.Merge(&instance, defaultInstance); err != nil {
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

func diskType(diskType nais.CloudSqlInstanceDiskType) string {
	return fmt.Sprintf("PD_%s", diskType)
}

func tier(cpu, memory int) string {
	return fmt.Sprintf("db-custom-%d-%d", cpu, memory)
}
