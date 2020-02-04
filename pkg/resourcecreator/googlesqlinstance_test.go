package resourcecreator_test

import (
	"fmt"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGoogleSqlInstance(t *testing.T) {
	app := fixtures.MinimalApplication()
	spec := nais.CloudSqlInstance{
		Name: app.Name,
		Type: "POSTGRES_11",
	}
	spec, err := resourcecreator.CloudSqlInstanceWithDefaults(spec, app.Name)
	assert.NoError(t, err)

	projectId := "projectid"
	sqlInstance := resourcecreator.GoogleSqlInstance(app, spec, projectId)
	assert.Equal(t, app.Name, sqlInstance.Name)
	assert.Equal(t, fmt.Sprintf("PD_%s", resourcecreator.DefaultSqlInstanceDiskType), sqlInstance.Spec.Settings.DiskType)
	assert.Equal(t, resourcecreator.DefaultSqlInstanceDiskSize, sqlInstance.Spec.Settings.DiskSize)
	assert.Equal(t, resourcecreator.DefaultSqlInstanceTier, sqlInstance.Spec.Settings.Tier)
	assert.Equal(t, projectId, sqlInstance.Annotations[resourcecreator.GoogleProjectIdAnnotation])
	assert.Equal(t, "02:00", sqlInstance.Spec.Settings.BackupConfiguration.StartTime)
	assert.True(t, sqlInstance.Spec.Settings.BackupConfiguration.Enabled)
	assert.Nil(t, sqlInstance.Spec.Settings.MaintenanceWindow, "user not specifying maintenance window leaves it unconfigured")

	t.Run("setting configuring maintenance and backup works as expected", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		backupHour := 0
		maintenanceDay := 6
		maintenanceHour := 9
		spec := nais.CloudSqlInstance{
			Name:           app.Name,
			Type:           "POSTGRES_11",
			AutoBackupHour: &backupHour,
			Maintenance: &nais.Maintenance{
				Day:  maintenanceDay,
				Hour: &maintenanceHour,
			},
		}
		spec, err := resourcecreator.CloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		sqlInstance := resourcecreator.GoogleSqlInstance(app, spec, projectId)
		assert.Equal(t, "00:00", sqlInstance.Spec.Settings.BackupConfiguration.StartTime, "setting backup hour to 0 yields 00:00 as start time")
		assert.Equal(t, maintenanceHour, sqlInstance.Spec.Settings.MaintenanceWindow.Hour)
		assert.Equal(t, maintenanceDay, sqlInstance.Spec.Settings.MaintenanceWindow.Day)
	})

}
