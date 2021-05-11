package google_storagebucket

import (
	"fmt"
	"time"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleStorageBucket(app *nais.Application, bucket nais.CloudStorageBucket) *google_storage_crd.StorageBucket {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = fmt.Sprintf("%s", bucket.Name)
	storagebucketPolicySpec := google_storage_crd.StorageBucketSpec{Location: google.GoogleRegion}

	if !bucket.CascadingDelete {
		util.SetAnnotation(&objectMeta, google.GoogleDeletionPolicyAnnotation, google.GoogleDeletionPolicyAbandon)
	}

	// Converting days to seconds if retention is set
	if bucket.RetentionPeriodDays != nil {
		retentionPeriod := *bucket.RetentionPeriodDays * int(time.Hour.Seconds()*24)
		storagebucketPolicySpec.RetentionPolicy = &google_storage_crd.RetentionPolicy{RetentionPeriod: retentionPeriod}
	}

	if bucket.LifecycleCondition != nil {
		lifecycleRule := google_storage_crd.LifecycleRules{
			Action: google_storage_crd.Action{Type: "Delete"},
			Condition: google_storage_crd.Condition{
				Age:              bucket.LifecycleCondition.Age,
				CreatedBefore:    bucket.LifecycleCondition.CreatedBefore,
				NumNewerVersions: bucket.LifecycleCondition.NumNewerVersions,
				WithState:        bucket.LifecycleCondition.WithState,
			},
		}
		storagebucketPolicySpec.LifecycleRules = append(storagebucketPolicySpec.LifecycleRules, lifecycleRule)
	}

	return &google_storage_crd.StorageBucket{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "StorageBucket",
			APIVersion: google.GoogleStorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec:       storagebucketPolicySpec,
	}
}

func StorageBucketIamPolicyMember(app *nais.Application, bucket *google_storage_crd.StorageBucket, googleProjectId, googleTeamProjectId string) *google_iam_crd.IAMPolicyMember {
	policyMemberName := fmt.Sprintf("%s-object-viewer", bucket.Name)
	policy := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: (*app).CreateObjectMetaWithName(policyMemberName),
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.GoogleIAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(app, googleProjectId)),
			Role:   "roles/storage.objectViewer",
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: bucket.APIVersion,
				Kind:       bucket.Kind,
				Name:       &bucket.Name,
			},
		},
	}

	util.SetAnnotation(policy, google.GoogleProjectIdAnnotation, googleTeamProjectId)

	return policy
}
