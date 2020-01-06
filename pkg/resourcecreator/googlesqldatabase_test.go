package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGoogleSqlDatabase(t *testing.T) {
	app := fixtures.MinimalApplication()
	databases := []nais.CloudSqlDatabase{{Name: "db1"}, {Name: "db2"}}
	sqlDatabases := resourcecreator.GoogleSqlDatabases(app, nais.CloudSqlInstance{Type: "POSTGRES_11", Databases: databases})
	assert.Equal(t, databases[0].Name, sqlDatabases[0].Name)
	assert.Equal(t, databases[1].Name, sqlDatabases[1].Name)
	assert.Len(t, sqlDatabases, len(databases))
	assert.Equal(t, app.Name, sqlDatabases[0].Spec.InstanceRef.Name)
	assert.Equal(t, resourcecreator.CascadingDeleteAnnotation(false), sqlDatabases[0].ObjectMeta.Annotations)
}
