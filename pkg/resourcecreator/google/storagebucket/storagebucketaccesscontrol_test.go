package google_storagebucket_test

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties/assert"
	storagebucket "github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
)

func TestGoogleStorageBucketAccessControl(t *testing.T) {
	t.Run("testing", func(t *testing.T) {
		bucketname := "buck-owens"
		projectId := "project-1234"
		serviceAccountName := "app-namespace-54203aa"
		app := fixtures.MinimalApplication()
		bac := storagebucket.AccessControl(resource.CreateObjectMeta(app), bucketname, projectId, serviceAccountName)

		assert.Equal(t, bac.Spec.BucketRef.Name, bucketname)
		assert.Equal(t, bac.Spec.Entity, fmt.Sprintf("user-%s@%s.iam.gserviceaccount.com", serviceAccountName, projectId))
		assert.Equal(t, bac.Spec.Role, "OWNER")
	})
}
