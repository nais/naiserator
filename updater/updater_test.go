package updater_test

import (
	"testing"

	google_storage_crd "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/nais/naiserator/pkg/util"
	"github.com/nais/naiserator/updater"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fixture() *google_storage_crd.StorageBucket {
	return &google_storage_crd.StorageBucket{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageBucket",
			APIVersion: google.StorageAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "namespace",
		},
		Spec: google_storage_crd.StorageBucketSpec{
			ResourceID: "resourceid",
		},
	}
}

// Test that existing annotations are copied over from Kubernetes
// to the newly created resource, but only if they match the CNRM prefix.
func TestCopyMeta(t *testing.T) {
	existing := fixture()
	updating := fixture()

	util.SetAnnotation(existing, "cnrm.cloud.google.com/existing-but-we-dont-care", "fooooo")
	util.SetAnnotation(existing, "cnrm.cloud.google.com/state-into-spec", "baz") // should be preserved
	util.SetAnnotation(existing, "foo", "oldvalue")

	util.SetAnnotation(updating, "foo", "newvalue")
	util.SetAnnotation(updating, "bar", "baz")

	err := updater.CopyImmutable(updating, existing)

	assert.NoError(t, err)
	assert.EqualValues(t, map[string]string{
		"foo":                                   "newvalue",
		"bar":                                   "baz",
		"cnrm.cloud.google.com/state-into-spec": "baz",
	}, updating.GetAnnotations())

	assert.Equal(t, "resourceid", updating.Spec.ResourceID)
}

func TestCopyAnnotation(t *testing.T) {
	expected := map[string]string{"foo": "bar"}
	src := fixtures.MinimalApplication()
	dst := fixtures.MinimalApplication()
	util.SetAnnotation(src, "foo", "bar")
	updater.CopyAnnotation(dst, src, "foo")
	assert.EqualValues(t, expected, dst.GetAnnotations())
	assert.EqualValues(t, expected, src.GetAnnotations())
}
