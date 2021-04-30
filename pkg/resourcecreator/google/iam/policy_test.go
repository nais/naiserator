package google_iam_test

import (
	"fmt"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/google/iam"
	"github.com/nais/naiserator/pkg/test/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestCreateGoogleIAMPolicy(t *testing.T) {
	app := fixtures.MinimalApplication()
	iamServiceAccount := google_iam.GoogleIAMServiceAccount(app, "")
	projectId := "nais-env-1234"
	iamPolicy := google_iam.GoogleIAMPolicy(app, &iamServiceAccount, projectId)
	testMember := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", projectId, app.Namespace, app.Name)
	assert.Equal(t, iamServiceAccount.Kind, iamPolicy.Spec.ResourceRef.Kind)
	assert.Equal(t, &iamServiceAccount.Name, iamPolicy.Spec.ResourceRef.Name)
	assert.Equal(t, "roles/iam.workloadIdentityUser", iamPolicy.Spec.Bindings[0].Role)
	assert.Equal(t, testMember, iamPolicy.Spec.Bindings[0].Members[0])
	assert.Equal(t, projectId, iamPolicy.Annotations[google.GoogleProjectIdAnnotation])
}
