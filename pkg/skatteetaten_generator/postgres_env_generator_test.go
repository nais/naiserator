package skatteetaten_generator

import (
	"testing"

	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

const (
	expectedURL = "testpostgresenv_URL"
	expectedDatasource = "jdbc:postgresql://${testpostgresenv_DATABASESERVER_NAME}.postgres.database.azure.com:5432/${testpostgresenv_DATABASE_NAME}?sslmode=require"
	prefix = "testpostgresenv"
	secretName = "secretName"
)


func TestPostgresEnv(t *testing.T) {
	t.Run("Postgres env should match prefix", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := app.ApplyDefaults()
		assert.NoError(t, err)
		envVars := GenerateDbEnv(prefix, secretName)
		assert.Equal(t, expectedURL, envVars[0].Name)
	})

	t.Run("Postgres env should match datasource", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		envVars := GenerateDbEnv(prefix, secretName)
		assert.Equal(t, expectedDatasource, envVars[0].Value)
	})


}
