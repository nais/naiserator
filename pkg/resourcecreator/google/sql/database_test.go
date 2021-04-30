package google_sql_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGoogleSqlDatabase(t *testing.T) {
	app := fixtures.MinimalApplication()
	database := nais.CloudSqlDatabase{Name: "db1"}
	instanceName := "instance-0"
	projectId := "projectid"
	sqlDatabase := google_sql.GoogleSQLDatabase(app, database, nais.CloudSqlInstance{Name: instanceName, Type: "POSTGRES_11"}, projectId)
	assert.Equal(t, database.Name, sqlDatabase.Name)
	assert.Equal(t, database.Name, sqlDatabase.Name)
	assert.Equal(t, instanceName, sqlDatabase.Spec.InstanceRef.Name)
	assert.Equal(t, google.GoogleDeletionPolicyAbandon, sqlDatabase.ObjectMeta.Annotations[google.GoogleDeletionPolicyAnnotation])
	assert.Equal(t, projectId, sqlDatabase.ObjectMeta.Annotations[google.GoogleProjectIdAnnotation])
}
