package resourcecreator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
)

func TestGetAuthorizationPolicy(t *testing.T) {
	otherApplication := "a"
	otherNamespace := "othernamespace"

	t.Run("auth policy", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{otherApplication, otherNamespace}}
		authorizationPolicy := resourcecreator.AuthorizationPolicy(app)
		assert.NotNil(t, authorizationPolicy)
	})
}
