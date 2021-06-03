package google_sql_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGoogleSqlDatabase(t *testing.T) {
	app := fixtures.MinimalApplication()
	database := nais.CloudSqlDatabase{Name: "db1"}
	instanceName := "instance-0"
	projectId := "projectid"
	sqlDatabase := google_sql.GoogleSQLDatabase(resource.CreateObjectMeta(app), database, nais.CloudSqlInstance{Name: instanceName, Type: "POSTGRES_11"}, projectId)
	assert.Equal(t, database.Name, sqlDatabase.Name)
	assert.Equal(t, database.Name, sqlDatabase.Name)
	assert.Equal(t, instanceName, sqlDatabase.Spec.InstanceRef.Name)
	assert.Equal(t, google.DeletionPolicyAbandon, sqlDatabase.ObjectMeta.Annotations[google.DeletionPolicyAnnotation])
	assert.Equal(t, projectId, sqlDatabase.ObjectMeta.Annotations[google.ProjectIdAnnotation])
}
