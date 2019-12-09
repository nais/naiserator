package resourcecreator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceAccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	svcAcc := resourcecreator.ServiceAccount(app, false, "")

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
}

func TestGetServiceAccountGoogleCluster(t *testing.T) {
	app := fixtures.MinimalApplication()
	svcAcc := resourcecreator.ServiceAccount(app, true, "nais-project-1234")

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
	assert.Equal(t, "myapplicati-mynamespac-w4o5cwa@nais-project-1234.iam.gserviceaccount.com", svcAcc.Annotations["iam.gke.io/gcp-service-account"])
}
