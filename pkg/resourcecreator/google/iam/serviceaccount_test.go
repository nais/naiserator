package google_iam_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestCreateGoogleIAMServiceaccount(t *testing.T) {
	app := fixtures.MinimalApplication()
	iamServiceAccount := google_iam.CreateServiceAccount(app)

	assert.Equal(t, app.Name, iamServiceAccount.Name)
}