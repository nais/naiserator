package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
)

func TestGoogleStorageBucketAccessControl(t *testing.T) {
	t.Run("testing", func(t *testing.T) {
		bucketname := "buck-owens"
		projectId := "project-1234"
		serviceAccountName := "app-namespace-54203aa"
		app := fixtures.MinimalApplication()
		bac := resourcecreator.GoogleStorageBucketAccessControl(app, bucketname, projectId, serviceAccountName)

		assert.Equal(t, bac.Spec.BucketRef.Name, bucketname)
		assert.Equal(t, bac.Spec.Entity, fmt.Sprintf("user-%s@%s.iam.gserviceaccount.com", serviceAccountName, projectId))
		assert.Equal(t, bac.Spec.Role, "OWNER")
	})
}
