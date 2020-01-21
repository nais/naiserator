package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleStorageBucket(app *nais.Application, bucket nais.CloudStorageBucket) *google_storage_crd.StorageBucket {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = fmt.Sprintf("%s", bucket.Name)

	if !bucket.CascadingDelete {
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}

	return &google_storage_crd.StorageBucket{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "StorageBucket",
			APIVersion: GoogleStorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_storage_crd.StorageBucketSpec{
			Location: GoogleRegion,
		},
	}
}
