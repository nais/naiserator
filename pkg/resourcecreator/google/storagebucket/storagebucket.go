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
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	google_storage_crd "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	objectUser   = "roles/storage.objectUser"
	objectViewer = "roles/storage.objectViewer"
)

type Source interface {
	resource.Source
	GetGCP() *nais_io_v1.GCP
}

type Config interface {
	GetGoogleProjectID() string
	GetGoogleTeamProjectID() string
}

func CreateBucket(objectMeta metav1.ObjectMeta, bucket nais_io_v1.CloudStorageBucket, projectId string) *google_storage_crd.StorageBucket {
	objectMeta.Name = bucket.Name
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)
	storagebucketPolicySpec := google_storage_crd.StorageBucketSpec{
		Location: google.Region,
		// Always enable uniform bucket-level access; ACLs are not used.
		UniformBucketLevelAccess: true,
		SoftDeletePolicy: &google_storage_crd.SoftDeletePolicy{
			RetentionDurationSeconds: 0,
		},
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
				Age:                 bucket.LifecycleCondition.Age,
				CreatedBefore:       bucket.LifecycleCondition.CreatedBefore,
				DaysSinceCustomTime: bucket.LifecycleCondition.DaysSinceCustomTime,
				NumNewerVersions:    bucket.LifecycleCondition.NumNewerVersions,
				WithState:           bucket.LifecycleCondition.WithState,
			},
		}
		storagebucketPolicySpec.LifecycleRules = append(storagebucketPolicySpec.LifecycleRules, lifecycleRule)
	}

	storagebucketPolicySpec.PublicAccessPrevention = google_storage_crd.PublicAccessPreventionInherited
	if bucket.PublicAccessPrevention {
		storagebucketPolicySpec.PublicAccessPrevention = google_storage_crd.PublicAccessPreventionEnforced
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

func iAMPolicyMember(source resource.Source, bucket *google_storage_crd.StorageBucket, cfg Config, role, policyNameSuffix string) (*google_iam_crd.IAMPolicyMember, error) {
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
			Member: fmt.Sprintf("serviceAccount:%s", google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), cfg.GetGoogleProjectID())),
			Role:   role,
			ResourceRef: google_iam_crd.ResourceRef{
				ApiVersion: bucket.APIVersion,
				Kind:       bucket.Kind,
				Name:       &bucket.Name,
			},
		},
	}

	util.SetAnnotation(policy, google.ProjectIdAnnotation, cfg.GetGoogleTeamProjectID())
	return policy, nil
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	gcp := source.GetGCP()
	if gcp == nil {
		return nil
	}

	for _, b := range gcp.Buckets {
		bucket := CreateBucket(resource.CreateObjectMeta(source), b, cfg.GetGoogleTeamProjectID())
		ast.AppendOperation(resource.OperationCreateOrUpdate, bucket)

		iamPolicyMember, err := iAMPolicyMember(source, bucket, cfg, objectUser, "object-user")

		if err != nil {
			return err
		}

		ast.AppendOperation(resource.OperationCreateIfNotExists, iamPolicyMember)

		viewer, err := iAMPolicyMember(source, bucket, cfg, objectViewer, "object-viewer")
		if err != nil {
			return err
		}

		ast.AppendOperation(resource.OperationCreateIfNotExists, viewer)

	}

	return nil
}
