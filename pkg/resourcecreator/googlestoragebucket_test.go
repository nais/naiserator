package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetGoogleStorageBucket(t *testing.T) {
	t.Run("bucket creation fails because of existing bucket", func(t *testing.T) {
		bucketname := "aura-test-bucket"
		app := fixtures.MinimalApplication()
		bucket, err := resourcecreator.GoogleStorageBucket(app, bucketname)

		assert.Nil(t, bucket)
		assert.Equal(t, fmt.Sprintf("bucket name '%s' is not available", bucketname), err.Error())
	})

	t.Run("bucket creation", func(t *testing.T) {
		bucketname := "buck-owens"
		app := fixtures.MinimalApplication()
		bucket, err := resourcecreator.GoogleStorageBucket(app, bucketname)
		assert.Nil(t, err)
		assert.Equal(t, "buck-owens", bucket.Name)
		assert.Equal(t, resourcecreator.GoogleRegion, bucket.Spec.Location)
	})
}
