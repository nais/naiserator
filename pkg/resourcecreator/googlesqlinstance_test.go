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

	sqlInstance, err := resourcecreator.GoogleSqlInstance(app, spec)
	assert.NoError(t, err)
	assert.Equal(t, app.Name, sqlInstance.Name)
	assert.Equal(t, fmt.Sprintf("PD_%s", resourcecreator.DefaultSqlInstanceDiskType), sqlInstance.Spec.Settings.DiskType)
	assert.Equal(t, resourcecreator.DefaultSqlInstanceDiskSize, sqlInstance.Spec.Settings.DiskSize)
	assert.Equal(t, resourcecreator.DefaultSqlInstanceTier, sqlInstance.Spec.Settings.Tier)
}
