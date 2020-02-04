package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestCreateGoogleIAMPolicy(t *testing.T) {
	app := fixtures.MinimalApplication()
	iamServiceAccount := resourcecreator.GoogleIAMServiceAccount(app, "")
	projectId := "nais-env-1234"
	iamPolicy := resourcecreator.GoogleIAMPolicy(app, &iamServiceAccount, projectId)
	testMember := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", projectId, app.Namespace, app.Name)
	assert.Equal(t, iamServiceAccount.Kind, iamPolicy.Spec.ResourceRef.Kind)
	assert.Equal(t, iamServiceAccount.Name, iamPolicy.Spec.ResourceRef.Name)
	assert.Equal(t, "roles/iam.workloadIdentityUser", iamPolicy.Spec.Bindings[0].Role)
	assert.Equal(t, testMember, iamPolicy.Spec.Bindings[0].Members[0])
	assert.Equal(t, projectId, iamPolicy.Annotations[resourcecreator.GoogleProjectIdAnnotation])
}
