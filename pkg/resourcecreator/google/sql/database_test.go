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
	projectID := "projectid"
	instance := nais.CloudSqlInstance{
		Name: "instance-0",
		Type: "POSTGRES_11",
	}
	database := nais.CloudSqlDatabase{
		Name: "db1",
	}
	sqlDatabase := google_sql.CreateGoogleSQLDatabase(
		resource.CreateObjectMeta(app),
		instance.Name,
		database.Name,
		projectID,
		instance.CascadingDelete,
	)

	assert.Equal(t, database.Name, sqlDatabase.Name)
	assert.Equal(t, database.Name, sqlDatabase.Name)
	assert.Equal(t, instance.Name, sqlDatabase.Spec.InstanceRef.Name)
	assert.Equal(t, google.DeletionPolicyAbandon, sqlDatabase.ObjectMeta.Annotations[google.DeletionPolicyAnnotation])
	assert.Equal(t, projectID, sqlDatabase.ObjectMeta.Annotations[google.ProjectIdAnnotation])
}
