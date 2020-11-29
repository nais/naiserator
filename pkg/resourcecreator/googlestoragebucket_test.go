package resourcecreator_test

import (
	"testing"
	"time"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetGoogleStorageBucket(t *testing.T) {
	t.Run("bucket creation", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		csb := nais.CloudStorageBucket{Name: "mystoragebucket", RetentionPeriodDays: intp(7)}
		expectedRetentionPeriod := *csb.RetentionPeriodDays * int(time.Hour.Seconds()*24)

		bucket := resourcecreator.GoogleStorageBucket(app, csb)
		assert.Equal(t, csb.Name, bucket.Name)
		assert.Equal(t, expectedRetentionPeriod, bucket.Spec.RetentionPolicy.RetentionPeriod)
		assert.Equal(t, resourcecreator.GoogleRegion, bucket.Spec.Location)
		assert.Equal(t, resourcecreator.GoogleDeletionPolicyAbandon, bucket.ObjectMeta.Annotations[resourcecreator.
			GoogleDeletionPolicyAnnotation])
	})

	t.Run("bucket without retention", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		csb := nais.CloudStorageBucket{Name: "mystoragebucket"}

		bucket := resourcecreator.GoogleStorageBucket(app, csb)
		assert.Equal(t, csb.Name, bucket.Name)
		assert.Nil(t, bucket.Spec.RetentionPolicy)
		assert.Equal(t, resourcecreator.GoogleRegion, bucket.Spec.Location)
		assert.Equal(t, resourcecreator.GoogleDeletionPolicyAbandon, bucket.ObjectMeta.Annotations[resourcecreator.
			GoogleDeletionPolicyAnnotation])
	})
}

func intp(i int) *int {
	return &i
}
