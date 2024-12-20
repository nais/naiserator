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
	util.SetAnnotation(existing, "foo", "oldvalue")

	util.SetAnnotation(updating, "foo", "newvalue")
	util.SetAnnotation(updating, "bar", "baz")

	err := updater.CopyImmutable(updating, existing)

	assert.NoError(t, err)
	assert.EqualValues(t, map[string]string{
		"foo": "newvalue",
		"bar": "baz",
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

func TestAssertOwnerReferenceEqual(t *testing.T) {
	multiOwnership := fixtures.MinimalApplication()
	ownedByCandidate := fixtures.MinimalApplication()
	ownedBySomethingElse := fixtures.MinimalApplication()
	notOwned := fixtures.MinimalApplication()
	resource := fixtures.MinimalApplication()

	ownedByCandidate.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "myapplication",
		},
	}

	ownedBySomethingElse.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "otherapplication", // different
		},
	}

	multiOwnership.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "myapplication",
		},
		{
			Kind: "Application",
			Name: "otherapplication",
		},
	}

	resource.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "myapplication",
		},
	}

	assert.NoError(t, updater.AssertOwnerReferenceEqual(resource, ownedByCandidate))
	assert.Error(t, updater.AssertOwnerReferenceEqual(resource, ownedBySomethingElse))
	assert.Error(t, updater.AssertOwnerReferenceEqual(resource, notOwned))
}

func TestKeepOwnerReferenceMultiOwner(t *testing.T) {
	resource := fixtures.MinimalApplication()
	existing := fixtures.MinimalApplication()

	resource.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "myapplication",
		},
	}

	existing.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "myapplication",
		},
		{
			Kind: "Application",
			Name: "otherapplication",
		},
	}

	assert.NoError(t, updater.KeepOwnerReference(resource, existing))

	assert.Len(t, resource.OwnerReferences, 2)
	assert.Equal(t, "myapplication", resource.OwnerReferences[0].Name)
	assert.Equal(t, "otherapplication", resource.OwnerReferences[1].Name)
}

func TestKeepOwnerReferenceNoOwner(t *testing.T) {
	resource := fixtures.MinimalApplication()
	existing := fixtures.MinimalApplication()

	resource.OwnerReferences = []metav1.OwnerReference{
		{
			Kind: "Application",
			Name: "myapplication",
		},
	}

	assert.NoError(t, updater.KeepOwnerReference(resource, existing))

	assert.Len(t, resource.OwnerReferences, 1)
	assert.Equal(t, "myapplication", resource.OwnerReferences[0].Name)
}