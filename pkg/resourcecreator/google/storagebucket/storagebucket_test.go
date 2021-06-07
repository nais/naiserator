package google_storagebucket_test

import (
	"testing"
	"time"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	storagebucket "github.com/nais/naiserator/pkg/resourcecreator/google/storagebucket"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetGoogleStorageBucket(t *testing.T) {
	t.Run("bucket creation", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		csb := nais.CloudStorageBucket{Name: "mystoragebucket"}

		bucket := storagebucket.CreateBucket(resource.CreateObjectMeta(app), csb)
		assert.Equal(t, csb.Name, bucket.Name)
		assert.Equal(t, google.Region, bucket.Spec.Location)
		assert.Equal(t, google.DeletionPolicyAbandon, bucket.ObjectMeta.Annotations[google.
			DeletionPolicyAnnotation])
		assert.Nil(t, bucket.Spec.RetentionPolicy)
		assert.Nil(t, bucket.Spec.LifecycleRules)
	})

	t.Run("bucket creation with retention", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		csb := nais.CloudStorageBucket{Name: "mystoragebucket", RetentionPeriodDays: intp(7)}
		expectedRetentionPeriod := *csb.RetentionPeriodDays * int(time.Hour.Seconds()*24)

		bucket := storagebucket.CreateBucket(resource.CreateObjectMeta(app), csb)
		assert.Equal(t, csb.Name, bucket.Name)
		assert.Equal(t, expectedRetentionPeriod, bucket.Spec.RetentionPolicy.RetentionPeriod)
		assert.Equal(t, google.Region, bucket.Spec.Location)
		assert.Equal(t, google.DeletionPolicyAbandon, bucket.ObjectMeta.Annotations[google.
			DeletionPolicyAnnotation])
		assert.Nil(t, bucket.Spec.LifecycleRules)
	})

	t.Run("bucket with life cycle rules", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		lifecycleCondition := nais.LifecycleCondition{
			Age:              7,
			CreatedBefore:    "2019-01-01",
			NumNewerVersions: 2,
			WithState:        "ANY",
		}
		csb := nais.CloudStorageBucket{Name: "mystoragebucket", LifecycleCondition: &lifecycleCondition}
		bucket := storagebucket.CreateBucket(resource.CreateObjectMeta(app), csb)
		lifecycleRule := bucket.Spec.LifecycleRules[0]

		assert.Equal(t, csb.Name, bucket.Name)
		assert.Equal(t, csb.LifecycleCondition.Age, lifecycleRule.Condition.Age)
		assert.Equal(t, csb.LifecycleCondition.CreatedBefore, lifecycleRule.Condition.CreatedBefore)
		assert.Equal(t, csb.LifecycleCondition.NumNewerVersions, lifecycleRule.Condition.NumNewerVersions)
		assert.Equal(t, csb.LifecycleCondition.WithState, lifecycleRule.Condition.WithState)
		assert.Equal(t, google.Region, bucket.Spec.Location)
		assert.Equal(t, google.DeletionPolicyAbandon, bucket.ObjectMeta.Annotations[google.
			DeletionPolicyAnnotation])
		assert.Nil(t, bucket.Spec.RetentionPolicy)
	})

}

func intp(i int) *int {
	return &i
}
