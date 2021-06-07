package google_storagebucket

import (
	"fmt"
	"time"

	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_storage_crd "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateBucket(objectMeta metav1.ObjectMeta, bucket nais.CloudStorageBucket) *google_storage_crd.StorageBucket {
	objectMeta.Name = fmt.Sprintf("%s", bucket.Name)
	storagebucketPolicySpec := google_storage_crd.StorageBucketSpec{Location: google.Region}

	if !bucket.CascadingDelete {
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageBucket",
			APIVersion: google.StorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec:       storagebucketPolicySpec,
	}
}

func iAMPolicyMember(source resource.Source, bucket *google_storage_crd.StorageBucket, googleProjectId, googleTeamProjectId string) *google_iam_crd.IAMPolicyMember {
	objectMeta := resource.CreateObjectMeta(source)
	policyMemberName := fmt.Sprintf("%s-object-viewer", bucket.Name)
	objectMeta.Name = policyMemberName
	policy := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), googleProjectId)),
			Role:   "roles/storage.objectViewer",
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: bucket.APIVersion,
				Kind:       bucket.Kind,
				Name:       &bucket.Name,
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, googleTeamProjectId)

	return policy
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, googleServiceAccount google_iam_crd.IAMServiceAccount, naisBucket []nais.CloudStorageBucket) {
	if naisBucket == nil {
		return
	}

	for _, b := range naisBucket {
		bucket := CreateBucket(resource.CreateObjectMeta(source), b)
		ast.AppendOperation(resource.OperationCreateIfNotExists, bucket)

		bucketAccessControl := AccessControl(resource.CreateObjectMeta(source), bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
		ast.AppendOperation(resource.OperationCreateOrUpdate, bucketAccessControl)

		iamPolicyMember := iAMPolicyMember(source, bucket, resourceOptions.GoogleProjectId, resourceOptions.GoogleTeamProjectId)
		ast.AppendOperation(resource.OperationCreateIfNotExists, iamPolicyMember)
	}
}
