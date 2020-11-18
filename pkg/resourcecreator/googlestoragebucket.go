package resourcecreator

import (
	"fmt"
	"time"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleStorageBucket(app *nais.Application, bucket nais.CloudStorageBucket) *google_storage_crd.StorageBucket {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = fmt.Sprintf("%s", bucket.Name)
	storageBucketSpec := google_storage_crd.StorageBucketSpec{Location: GoogleRegion}

	if !bucket.CascadingDelete {
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}

	// Converting days to seconds if retention is set
	if bucket.RetentionPeriodDays != 0 {
		retentionPeriod := bucket.RetentionPeriodDays * int(time.Hour.Seconds()*24)
		storageBucketSpec = google_storage_crd.StorageBucketSpec{
			Location:        GoogleRegion,
			RetentionPolicy: google_storage_crd.RetentionPolicy{RetentionPeriod: retentionPeriod},
		}
	}

	return &google_storage_crd.StorageBucket{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "StorageBucket",
			APIVersion: GoogleStorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec:       storageBucketSpec,
	}
}
