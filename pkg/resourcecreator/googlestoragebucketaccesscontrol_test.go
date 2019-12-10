package resourcecreator_test

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
)

func TestGoogleStorageBucketAccessControl(t *testing.T) {
	t.Run("testing", func(t *testing.T) {
		bucketname := "buck-owens"
		app := fixtures.MinimalApplication()
		bucket, _ := resourcecreator.GoogleStorageBucket(app, bucketname)
		googleSvcAcc := resourcecreator.GoogleServiceAccount(app)
		bac := resourcecreator.GoogleStorageBucketAccessControl(app,bucket,&googleSvcAcc)

		assert.Equal(t, bac.Spec.BucketRef.Name, bucketname)
		assert.Equal(t, bac.Spec.Entity, googleSvcAcc.Spec.DisplayName)
		assert.Equal(t, bac.Spec.Role, "OWNER")
	})
}
