package google_storagebucket

import (
	"fmt"

	google_storage_crd "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AccessControl(objectMeta metav1.ObjectMeta, bucketName, projectId, serviceAccountName string) *google_storage_crd.StorageBucketAccessControl {
	objectMeta.Name = bucketName
	util.SetAnnotation(&objectMeta, google.StateIntoSpec, google.StateIntoSpecValue)

	return &google_storage_crd.StorageBucketAccessControl{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageBucketAccessControl",
			APIVersion: google.StorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_storage_crd.StorageBucketAccessControlSpec{
			BucketRef: google_storage_crd.BucketRef{
				Name: bucketName,
			},
			Entity: fmt.Sprintf("user-%s@%s.iam.gserviceaccount.com", serviceAccountName, projectId),
			Role:   "OWNER",
		},
	}
}
