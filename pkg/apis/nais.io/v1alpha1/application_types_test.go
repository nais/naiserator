package v1alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Change this value to accept re-synchronization of ALL application resources when deploying a new version.
	applicationHash = "ba79fcde22026d1d"
)

func TestApplication_Hash(t *testing.T) {
	apps := []*v1alpha1.Application{
		{Spec: v1alpha1.ApplicationSpec{}},
		{Spec: v1alpha1.ApplicationSpec{}, ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"a": "b", "team": "banan"}}},
		{Spec: v1alpha1.ApplicationSpec{}, ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"a": "b", "team": "banan"}}},
	}
	hashes := make([]string, len(apps))
	for i := range apps {
		err := v1alpha1.ApplyDefaults(apps[i])
		if err != nil {
			panic(err)
		}
		hashes[i], err = apps[i].Hash()
		if err != nil {
			panic(err)
		}
	}

	assert.Equal(t, hashes[0], hashes[1], "matches, as annotations is ignored")
	assert.NotEqual(t, hashes[1], hashes[2], "should not match")
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

// Test that updating the application spec with new, default-null values does not trigger a hash change.
func TestHashJSONMarshalling(t *testing.T) {
	type a struct {
		Foo string `json:"foo"`
	}
	type oldspec struct {
		A *a `json:"a,omitempty"`
	}
	type newspec struct {
		A *a     `json:"a,omitempty"`
		B *a     `json:"b,omitempty"` // new field added to crd spec
		C string `json:"c,omitempty"` // new field added to crd spec
	}
	old := &oldspec{}
	neu := &newspec{}
	oldMarshal, _ := json.Marshal(old)
	newMarshal, _ := json.Marshal(neu)
	oldHash, _ := hashstructure.Hash(oldMarshal, nil)
	newHash, _ := hashstructure.Hash(newMarshal, nil)
	assert.Equal(t, newHash, oldHash)
}

func TestNewCRD(t *testing.T) {
	app := &v1alpha1.Application{}
	err := v1alpha1.ApplyDefaults(app)
	if err != nil {
		panic(err)
	}
	hash, err := app.Hash()
	assert.NoError(t, err)
	assert.Equalf(t, applicationHash, hash, "Your Application default value changes will trigger a FULL REDEPLOY of ALL APPLICATIONS in ALL NAMESPACES across ALL CLUSTERS. If this is what you really want, change the `applicationHash` constant in this test file to `%s`.", hash)
}

func TestAddAccessPolicyExternalHosts(t *testing.T) {
	app := &v1alpha1.Application{}
	err := v1alpha1.ApplyDefaults(app)
	assert.NoError(t, err)

	hosts := []string{
		"some-host.example.test",
		"existing.host.test",
	}
	app.Spec.AccessPolicy.Outbound.External = append(app.Spec.AccessPolicy.Outbound.External,
		v1alpha1.AccessPolicyExternalRule{Host: "existing.host.test"},
	)
	app.AddAccessPolicyExternalHosts(hosts)

	assert.Len(t, app.Spec.AccessPolicy.Outbound.External, 2)
	assert.Contains(t, app.Spec.AccessPolicy.Outbound.External, v1alpha1.AccessPolicyExternalRule{Host: "existing.host.test"})
	assert.Contains(t, app.Spec.AccessPolicy.Outbound.External, v1alpha1.AccessPolicyExternalRule{Host: "some-host.example.test"})
}
