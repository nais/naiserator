package resourcecreator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestGetGoogleServiceAccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	googleSvcAcc := resourcecreator.GoogleServiceAccount(app)

	assert.Equal(t, app.CreateAppNamespaceHash(), googleSvcAcc.Name)
	assert.Equal(t, resourcecreator.GoogleIAMServiceAccountNamespace, googleSvcAcc.Namespace)
}
