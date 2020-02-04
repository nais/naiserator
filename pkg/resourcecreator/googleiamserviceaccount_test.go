package resourcecreator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestCreateGoogleIAMServiceaccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	projectId := "projectId"
	iamServiceAccount := resourcecreator.GoogleIAMServiceAccount(app, projectId)

	assert.Equal(t, app.CreateAppNamespaceHash(), iamServiceAccount.Name)
	assert.Equal(t, resourcecreator.GoogleIAMServiceAccountNamespace, iamServiceAccount.Namespace)
	assert.Equal(t, projectId, iamServiceAccount.Annotations[resourcecreator.GoogleProjectIdAnnotation])
}
