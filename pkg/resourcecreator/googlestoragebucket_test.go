package resourcecreator_test

import (
	"fmt"
	"testing"

	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetGoogleStorageBuckets(t *testing.T) {
	t.Run("bucket creation fails because of existing bucket", func(t *testing.T) {
		bucketname := "aura-test-bucket"
		app := fixtures.MinimalApplication()
		app.Spec.ObjectStorage = []v1alpha1.ObjectStorage{
			v1alpha1.ObjectStorage{Name: bucketname},
		}

		bucket, err := resourcecreator.GoogleStorageBuckets(app)

		assert.Nil(t, bucket)
		assert.Equal(t, fmt.Sprintf("bucket name '%s' is not available", bucketname), err.Error())
	})

	t.Run("bucket creation", func(t *testing.T) {
		bucketname := "buck-owens"
		app := fixtures.MinimalApplication()
		app.Spec.ObjectStorage = []v1alpha1.ObjectStorage{
			v1alpha1.ObjectStorage{Name: bucketname},
		}
		bucket, err := resourcecreator.GoogleStorageBuckets(app)
		assert.Nil(t, err)
		assert.Equal(t, "buck-owens", bucket[0].Name)
		assert.Equal(t, resourcecreator.GoogleRegion, bucket[0].Spec.Location)
	})
}
