package v1alpha1_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplication_Hash(t *testing.T) {
	a1, err := v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{}}.Hash()
	a2, _ := v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{}, ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"a": "b", "team": "banan"}}}.Hash()
	a3, _ := v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{}, ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"a": "b", "team": "banan"}}}.Hash()

	assert.NoError(t, err)
	assert.Equal(t, a1, a2, "matches, as annotations is ignored")
	assert.NotEqual(t, a2, a3, "must not match ")
}

func TestApplication_CreateAppNamespaceHash(t *testing.T) {
	application := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "reallylongapplicationname",
			Namespace: "evenlongernamespacename",
		},
	}
	application2 := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "reallylongafoo",
			Namespace: "evenlongerbar",
		},
	}
	application3 := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "short",
			Namespace: "name",
		},
	}
	appNameHash := application.CreateAppNamespaceHash()
	appNameHash2 := application2.CreateAppNamespaceHash()
	appNameHash3 := application3.CreateAppNamespaceHash()
	assert.Equal(t, "reallylonga-evenlonger-siqwsiq", appNameHash)
	assert.Equal(t, "reallylonga-evenlonger-piuqbfq", appNameHash2)
	assert.Equal(t, "short-name-hqb7npi", appNameHash3)
	assert.True(t, len(appNameHash2) <= 30)
	assert.True(t, len(appNameHash3) >= 6)
}
