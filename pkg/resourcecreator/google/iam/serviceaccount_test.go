package google_iam_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestCreateGoogleIAMServiceaccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	projectId := "projectId"
	iamServiceAccount := google_iam.CreateServiceAccount(app.CreateObjectMeta(), projectId, app.CreateAppNamespaceHash())

	assert.Equal(t, app.CreateAppNamespaceHash(), iamServiceAccount.Name)
	assert.Equal(t, google.IAMServiceAccountNamespace, iamServiceAccount.Namespace)
	assert.Equal(t, projectId, iamServiceAccount.Annotations[google.ProjectIdAnnotation])
}
