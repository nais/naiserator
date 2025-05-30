package google_sql_test

import (
	"fmt"
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/nais/naiserator/pkg/util"
	"github.com/stretchr/testify/assert"
)

type MockConfig struct {
	GoogleProjectID                   string
	GoogleTeamProjectID               string
	GoogleCloudSQLProxyContainerImage string
	CreateSqlInstanceInSharedVpc      bool
	sqlInstanceExists                 bool
	sqlInstanceHasPrivateIp           bool
	clusterName                       string
}

func (m *MockConfig) GetGoogleProjectID() string {
	return m.GoogleProjectID
}

func (m *MockConfig) GetGoogleTeamProjectID() string {
	return m.GoogleTeamProjectID
}

func (m *MockConfig) GetGoogleCloudSQLProxyContainerImage() string {
	return m.GoogleCloudSQLProxyContainerImage
}

func (m *MockConfig) ShouldCreateSqlInstanceInSharedVpc() bool {
	return m.CreateSqlInstanceInSharedVpc
}

func (m *MockConfig) SqlInstanceExists() bool {
	return m.sqlInstanceExists
}

func (m *MockConfig) SqlInstanceHasPrivateIpInSharedVpc() bool {
	return m.sqlInstanceHasPrivateIp
}

func (m *MockConfig) GetClusterName() string {
	return m.clusterName
}

func TestGoogleSqlInstance(t *testing.T) {
	const tier = "db-custom-1-3840"

	app := fixtures.MinimalApplication()

	cfg := &MockConfig{
		GoogleProjectID:                   "clusterProjectId",
		GoogleTeamProjectID:               "teamProjectId",
		GoogleCloudSQLProxyContainerImage: "cloudsql/image:latest",
		CreateSqlInstanceInSharedVpc:      true,
		sqlInstanceExists:                 false,
		clusterName:                       "test-cluster",
	}

	spec := &nais.CloudSqlInstance{
		Name: app.Name,
		Type: "POSTGRES_17",
		Tier: tier,
	}
	spec, err := google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
	assert.NoError(t, err)

	googleSqlInstance, err := google_sql.CreateGoogleSqlInstance(resource.CreateObjectMeta(app), spec, cfg)
	assert.NoError(t, err)
	assert.Equal(t, app.Name, googleSqlInstance.Name)
	assert.Equal(t, fmt.Sprintf("PD_%s", google_sql.DefaultSqlInstanceDiskType), googleSqlInstance.Spec.Settings.DiskType)
	assert.Equal(t, google_sql.DefaultSqlInstanceDiskSize, googleSqlInstance.Spec.Settings.DiskSize)
	assert.Equal(t, tier, googleSqlInstance.Spec.Settings.Tier)
	assert.Equal(t, cfg.GoogleTeamProjectID, googleSqlInstance.Annotations[google.ProjectIdAnnotation])
	assert.Equal(t, "02:00", googleSqlInstance.Spec.Settings.BackupConfiguration.StartTime)
	assert.True(t, googleSqlInstance.Spec.Settings.BackupConfiguration.Enabled)
	assert.True(t, googleSqlInstance.Spec.Settings.IpConfiguration.RequireSsl)
	assert.Nil(t, googleSqlInstance.Spec.Settings.MaintenanceWindow, "user not specifying maintenance window leaves it unconfigured")

	t.Run("setting configuring maintenance and backup works as expected", func(t *testing.T) {
		const backupHour = 0
		const maintenanceDay = 6
		const maintenanceHour = 9

		app := fixtures.MinimalApplication()
		spec := &nais.CloudSqlInstance{
			Name:           app.Name,
			Type:           nais.CloudSqlInstanceTypePostgres12,
			AutoBackupHour: util.Intp(backupHour),
			Tier:           tier,
			Maintenance: &nais.Maintenance{
				Day:  maintenanceDay,
				Hour: util.Intp(maintenanceHour),
			},
		}
		spec, err := google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		googleSqlInstance, err := google_sql.CreateGoogleSqlInstance(resource.CreateObjectMeta(app), spec, cfg)
		assert.NoError(t, err)
		assert.Equal(t, "00:00", googleSqlInstance.Spec.Settings.BackupConfiguration.StartTime, "setting backup hour to 0 yields 00:00 as start time")
		assert.Equal(t, maintenanceHour, googleSqlInstance.Spec.Settings.MaintenanceWindow.Hour)
		assert.Equal(t, maintenanceDay, googleSqlInstance.Spec.Settings.MaintenanceWindow.Day)
	})

	t.Run("instance name is setInstanceName, defaults should not override", func(t *testing.T) {
		const naisSpecConfiguredInstanceName = "my-instance"

		app := fixtures.MinimalApplication()

		spec := &nais.CloudSqlInstance{
			Name: naisSpecConfiguredInstanceName,
			Type: nais.CloudSqlInstanceTypePostgres12,
			Tier: tier,
		}

		spec, err = google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		assert.Equal(t, naisSpecConfiguredInstanceName, spec.Name)
	})

	t.Run("instance name is not setInstanceName, defaults should be used for instance name", func(t *testing.T) {
		app := fixtures.MinimalApplication()

		spec := &nais.CloudSqlInstance{
			Type: nais.CloudSqlInstanceTypePostgres12,
			Tier: tier,
		}

		spec, err = google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		assert.Equal(t, app.Name, spec.Name)
	})

	t.Run("disk size is updated", func(t *testing.T) {
		const alternateDiskSize = 20

		app := fixtures.MinimalApplication()

		spec := &nais.CloudSqlInstance{
			DiskSize: alternateDiskSize,
			Tier:     tier,
		}

		spec, err = google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		googleSqlInstance, err := google_sql.CreateGoogleSqlInstance(resource.CreateObjectMeta(app), spec, cfg)
		assert.NoError(t, err)
		assert.Equal(t, googleSqlInstance.Spec.Settings.DiskSize, alternateDiskSize)
	})

	t.Run("private ip not applied when sqlinstance exists", func(t *testing.T) {
		cfg.sqlInstanceExists = true
		cfg.sqlInstanceHasPrivateIp = false

		app := fixtures.MinimalApplication()

		spec := &nais.CloudSqlInstance{
			Tier: tier,
		}

		spec, err = google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		googleSqlInstance, err := google_sql.CreateGoogleSqlInstance(resource.CreateObjectMeta(app), spec, cfg)
		assert.NoError(t, err)
		assert.Nil(t, googleSqlInstance.Spec.Settings.IpConfiguration.PrivateNetworkRef)
	})

	t.Run("private ip is applied when sqlinstance exists and already has private ip", func(t *testing.T) {
		cfg.sqlInstanceExists = true
		cfg.sqlInstanceHasPrivateIp = true

		app := fixtures.MinimalApplication()

		spec := &nais.CloudSqlInstance{
			Tier: tier,
		}

		spec, err = google_sql.NaisCloudSqlInstanceWithDefaults(spec, app.Name)
		assert.NoError(t, err)
		googleSqlInstance, err := google_sql.CreateGoogleSqlInstance(resource.CreateObjectMeta(app), spec, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, googleSqlInstance.Spec.Settings.IpConfiguration.PrivateNetworkRef)
	})

	t.Run("rollout refused when more than one sql instance requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.GCP = &nais.GCP{
			SqlInstances: []nais.CloudSqlInstance{
				{
					Type: "POSTGRES_17",
					Name: "postgres-11",
					Tier: tier,
				},
				{
					Type: "POSTGRES_17",
					Name: "postgres-12",
					Tier: tier,
				},
			},
		}

		ast := resource.NewAst()
		err := google_sql.CreateInstance(app, ast, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only one sql instance is supported")
	})

	t.Run("rollout refused when more than one sql database requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.GCP = &nais.GCP{
			SqlInstances: []nais.CloudSqlInstance{
				{
					Type: "POSTGRES_15",
					Name: "postgres-15",
					Tier: tier,
					Databases: []nais.CloudSqlDatabase{
						{
							Name: "db1",
							Users: []nais.CloudSqlDatabaseUser{
								{Name: "user1"},
							},
						},
						{
							Name: "db2",
							Users: []nais.CloudSqlDatabaseUser{
								{Name: "user2"},
							},
						},
					},
				},
			},
		}

		ast := resource.NewAst()
		err := google_sql.CreateInstance(app, ast, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only one sql database is supported")
	})
}
