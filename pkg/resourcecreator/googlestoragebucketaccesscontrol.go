package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleStorageBucketAccessControl(app *nais.Application, bucketName, projectId, serviceAccountName string) *google_storage_crd.StorageBucketAccessControl {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = bucketName

	return &google_storage_crd.StorageBucketAccessControl{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "StorageBucketAccessControl",
			APIVersion: GoogleStorageAPIVersion,
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
