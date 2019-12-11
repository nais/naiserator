package resourcecreator

import (
	"fmt"
	"net/http"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1alpha2"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validateBucketName(bucketname string) error {
	urlString := fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s", bucketname)
	resp, _ := http.Get(urlString)

	if resp.StatusCode != 404 {
		return fmt.Errorf("bucket name '%s' is not available", bucketname)
	}
	return nil
}
func GoogleStorageBuckets(app *nais.Application) (googleBuckets []*google_storage_crd.GoogleStorageBucket, err error) {
	for _, bucket := range app.Spec.CloudStorage {
		err := validateBucketName(bucket.Name)
		if err != nil {
			// TODO: return error event
		}
		googleBuckets = append(googleBuckets, GoogleStorageBucket(app, bucket.Name))
	}
	return googleBuckets, err

}
func GoogleStorageBucket(app *nais.Application, bucketName string) *google_storage_crd.GoogleStorageBucket {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = bucketName

	return &google_storage_crd.GoogleStorageBucket{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "StorageBucket",
			APIVersion: GoogleStorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_storage_crd.GoogleStorageBucketSpec{
			Location: GoogleRegion,
		},
	}
}
