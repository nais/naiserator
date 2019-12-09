package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestGetGoogleServiceAccountBinding(t *testing.T) {
	app := fixtures.MinimalApplication()
	googleSvcAcc := resourcecreator.GoogleServiceAccount(app)
	googleSvcAccBinding := resourcecreator.GoogleServiceAccountBinding(app, &googleSvcAcc, "nais-env-1234")
	testMember := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", "nais-env-1234", app.Namespace, app.Name)
	assert.Equal(t, googleSvcAcc.Kind, googleSvcAccBinding.Spec.ResourceRef.Kind)
	assert.Equal(t, googleSvcAcc.Name, googleSvcAccBinding.Spec.ResourceRef.Name)
	assert.Equal(t, "roles/iam.workloadIdentityUser", googleSvcAccBinding.Spec.Bindings[0].Role)
	assert.Equal(t, testMember, googleSvcAccBinding.Spec.Bindings[0].Members[0])
}
