package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleStorageBuckets(app *nais.Application) []*google_storage_crd.StorageBucket {
	googleBuckets := make([]*google_storage_crd.StorageBucket, len(app.Spec.GCP.Buckets))
	for i, bucket := range app.Spec.GCP.Buckets {
		googleBuckets[i] = GoogleStorageBucket(app, bucket.Name)
	}
	return googleBuckets

}

func GoogleStorageBucket(app *nais.Application, bucketName string) *google_storage_crd.StorageBucket {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = bucketName

	ApplyAbandonDeletionPolicy(&objectMeta)

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
