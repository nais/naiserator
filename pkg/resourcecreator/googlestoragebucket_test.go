package resourcecreator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetGoogleStorageBuckets(t *testing.T) {
	t.Run("bucket creation", func(t *testing.T) {
		bucketname := "buck-owens"
		app := fixtures.MinimalApplication()
		app.Spec.GCP = &v1alpha1.GCP{
			Buckets: []v1alpha1.CloudStorageBucket{
				{Name: bucketname},
			},
		}
		bucket := resourcecreator.GoogleStorageBuckets(app)
		assert.Equal(t, "buck-owens", bucket[0].Name)
		assert.Equal(t, resourcecreator.GoogleRegion, bucket[0].Spec.Location)
	})
}
