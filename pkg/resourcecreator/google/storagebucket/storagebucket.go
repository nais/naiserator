package google_storagebucket

import (
	"fmt"
	"time"

	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	"k8s.io/apimachinery/pkg/util/validation"

	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_storage_crd "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	objectViewer      = "roles/storage.objectViewer"
	legacyObjectOwner = "roles/storage.legacyObjectOwner"
	legacyBucketOwner = "roles/storage.legacyBucketOwner"
)

func CreateBucket(objectMeta metav1.ObjectMeta, bucket nais.CloudStorageBucket, projectId string) *google_storage_crd.StorageBucket {
	objectMeta.Name = bucket.Name
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)
	util.SetAnnotation(&objectMeta, google.StateIntoSpec, google.StateIntoSpecValue)
	storagebucketPolicySpec := google_storage_crd.StorageBucketSpec{
		Location:                 google.Region,
		UniformBucketLevelAccess: bucket.UniformBucketLevelAccess,
	}

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

func iAMPolicyMember(source resource.Source, bucket *google_storage_crd.StorageBucket, resourceOptions resource.Options, role, policyNameSuffix string) (*google_iam_crd.IAMPolicyMember, error) {
	objectMeta := resource.CreateObjectMeta(source)
	policyMemberName, err := namegen.ShortName(fmt.Sprintf("%s-%s", bucket.Name, policyNameSuffix), validation.DNS1035LabelMaxLength)
	if err != nil {
		return nil, err
	}
	objectMeta.Name = policyMemberName
	policy := &google_iam_crd.IAMPolicyMember{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicyMember",
			APIVersion: google.IAMAPIVersion,
		},
		Spec: google_iam_crd.IAMPolicyMemberSpec{
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), resourceOptions.GoogleProjectId)),
			Role:   role,
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: bucket.APIVersion,
				Kind:       bucket.Kind,
				Name:       &bucket.Name,
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, resourceOptions.GoogleTeamProjectId)

	return policy, nil
}

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, googleServiceAccount google_iam_crd.IAMServiceAccount, naisBucket []nais.CloudStorageBucket) error {
	if naisBucket == nil {
		return nil
	}

	for _, b := range naisBucket {
		bucket := CreateBucket(resource.CreateObjectMeta(source), b, resourceOptions.GoogleTeamProjectId)
		ast.AppendOperation(resource.OperationCreateOrUpdate, bucket)

		if b.UniformBucketLevelAccess {
			bucketOwner, err := iAMPolicyMember(source, bucket, resourceOptions, legacyBucketOwner, "legacy-bucket-owner")
			if err != nil {
				return err
			}
			ast.AppendOperation(resource.OperationCreateIfNotExists, bucketOwner)
			objectOwner, err := iAMPolicyMember(source, bucket, resourceOptions, legacyObjectOwner, "legacy-object-owner")
			if err != nil {
				return err
			}
			ast.AppendOperation(resource.OperationCreateIfNotExists, objectOwner)
		} else {
			bucketAccessControl := AccessControl(resource.CreateObjectMeta(source), bucket.Name, resourceOptions.GoogleProjectId, googleServiceAccount.Name)
			ast.AppendOperation(resource.OperationCreateOrUpdate, bucketAccessControl)
		}

		iamPolicyMember, err := iAMPolicyMember(source, bucket, resourceOptions, objectViewer, "object-viewer")
		if err != nil {
			return err
		}
		ast.AppendOperation(resource.OperationCreateIfNotExists, iamPolicyMember)
	}
	return nil
}
