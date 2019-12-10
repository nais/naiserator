package resourcecreator

import (
	"fmt"

	"github.com/nais/naiserator/pkg/apis/iam.cnrm.cloud.google.com/v1alpha1"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/apis/storage.cnrm.cloud.google.com/v1alpha2"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GoogleStorageBucketAccessControl(app *nais.Application, bucket *v1alpha2.GoogleStorageBucket, sa *v1alpha1.IAMServiceAccount) (v1alpha2.GoogleStorageBucketAccessControl) {
	objectMeta := app.CreateObjectMeta()
	objectMeta.Namespace = app.Namespace
	objectMeta.Name = fmt.Sprintf("%s-bac", bucket.Name)

	return v1alpha2.GoogleStorageBucketAccessControl{
		TypeMeta:   k8s_meta.TypeMeta{
			Kind:       "StorageBucketAccessControl",
			APIVersion: GoogleStorageAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec:       v1alpha2.GoogleStorageBucketAccessControlSpec{
			BucketRef: v1alpha2.BucketRef{
				Name: bucket.Name,
			},
			// TODO: FRODE...
			Entity:    sa.Spec.DisplayName,
			Role:      "OWNER",
		},
	}
}