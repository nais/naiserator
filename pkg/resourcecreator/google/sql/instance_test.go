package google_sql_test

import (
	"fmt"
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGoogleSqlInstance(t *testing.T) {
	app := fixtures.MinimalApplication()
	spec := nais.CloudSqlInstance{
		Name: app.Name,
		Type: "POSTGRES_11",
	}
	spec, err := google_sql.CloudSqlInstanceWithDefaults(spec, app.Name)
	assert.NoError(t, err)

	projectId := "projectid"
	sqlInstance := google_sql.GoogleSqlInstance(resource.CreateObjectMeta(app), spec, projectId)
	assert.Equal(t, app.Name, sqlInstance.Name)
	assert.Equal(t, fmt.Sprintf("PD_%s", google_sql.DefaultSqlInstanceDiskType), sqlInstance.Spec.Settings.DiskType)
	assert.Equal(t, google_sql.DefaultSqlInstanceDiskSize, sqlInstance.Spec.Settings.DiskSize)
	assert.Equal(t, google_sql.DefaultSqlInstanceTier, sqlInstance.Spec.Settings.Tier)
	assert.Equal(t, projectId, sqlInstance.Annotations[google.ProjectIdAnnotation])
	assert.Equal(t, "02:00", sqlInstance.Spec.Settings.BackupConfiguration.StartTime)
	assert.True(t, sqlInstance.Spec.Settings.BackupConfiguration.Enabled)
	assert.True(t, sqlInstance.Spec.Settings.IpConfiguration.RequireSsl)
	assert.Nil(t, sqlInstance.Spec.Settings.MaintenanceWindow, "user not specifying maintenance window leaves it unconfigured")

	t.Run("setting configuring maintenance and backup works as expected", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		backupHour := 0
		maintenanceDay := 6
		maintenanceHour := 9
		spec := nais.CloudSqlInstance{
			Name:           app.Name,
			Type:           nais.CloudSqlInstanceTypePostgres12,
			AutoBackupHour: &backupHour,
			Maintenance: &nais.Maintenance{
				Day:  maintenanceDay,
				Hour: &maintenanceHour,
			},
		}
		spec, err := google_sql.CloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		sqlInstance := google_sql.GoogleSqlInstance(resource.CreateObjectMeta(app), spec, projectId)
		assert.Equal(t, "00:00", sqlInstance.Spec.Settings.BackupConfiguration.StartTime, "setting backup hour to 0 yields 00:00 as start time")
		assert.Equal(t, maintenanceHour, sqlInstance.Spec.Settings.MaintenanceWindow.Hour)
		assert.Equal(t, maintenanceDay, sqlInstance.Spec.Settings.MaintenanceWindow.Day)
	})

}
