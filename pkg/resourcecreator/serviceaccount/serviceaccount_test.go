package serviceaccount_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"

	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceAccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	opts := resource.NewOptions()
	svcAcc := serviceaccount.ServiceAccount(app, opts)

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
}

func TestGetServiceAccountGoogleCluster(t *testing.T) {
	app := fixtures.MinimalApplication()
	opts := resource.NewOptions()
	opts.GoogleProjectId = "nais-project-1234"
	svcAcc := serviceaccount.ServiceAccount(app, opts)

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
	assert.Equal(t, app.CreateAppNamespaceHash()+"@nais-project-1234.iam.gserviceaccount.com", svcAcc.Annotations["iam.gke.io/gcp-service-account"])
}
