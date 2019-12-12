package resourcecreator

import (
	"fmt"
	"net/http"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	google_storage_crd "github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1alpha2"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func validateBucketNames(app *nais.Application) error {
	for _, bucket := range app.Spec.ObjectStorage {
		bucketname := bucket.Name

		urlString := fmt.Sprintf("https://www.googleapis.com/storage/v1/b/%s", bucketname)
		resp, _ := http.Get(urlString)

		if resp.StatusCode != 404 {
			if




			return fmt.Errorf("bucket name '%s' is not available\n", bucketname)
		}
	}
	return nil
}
func GoogleStorageBuckets(app *nais.Application) (googleBuckets []*google_storage_crd.GoogleStorageBucket, err error) {
	err = validateBucketNames(app)
	if err != nil {
		return nil, err
	}
	for _, bucket := range app.Spec.ObjectStorage {
		googleBuckets = append(googleBuckets, GoogleStorageBucket(app, bucket.Name))
	}
	return googleBuckets, nil

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
