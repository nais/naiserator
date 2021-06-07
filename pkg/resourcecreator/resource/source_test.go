package resource

import (
	"reflect"
	"testing"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplication_CreateAppNamespaceHash(t *testing.T) {
	application := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reallylongapplicationname",
			Namespace: "evenlongernamespacename",
		},
	}
	application2 := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reallylongafoo",
			Namespace: "evenlongerbar",
		},
	}
	application3 := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "short",
			Namespace: "name",
		},
	}
	appNameHash := CreateAppNamespaceHash(application)
	appNameHash2 := CreateAppNamespaceHash(application2)
	appNameHash3 := CreateAppNamespaceHash(application3)
	assert.Equal(t, "reallylonga-evenlonger-siqwsiq", appNameHash)
	assert.Equal(t, "reallylonga-evenlonger-piuqbfq", appNameHash2)
	assert.Equal(t, "short-name-hqb7npi", appNameHash3)
	assert.True(t, len(appNameHash2) <= 30)
	assert.True(t, len(appNameHash3) >= 6)
}

func TestApplication_CreateObjectMeta(t *testing.T) {
	const app, namespace, team, key, value = "myapp", "mynamespace", "myteam", "key", "value"

	tests := []struct {
		name string
		in   *nais_io_v1alpha1.Application
		want map[string]string
	}{
		{
			"test object meta plain",
			&nais_io_v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels: map[string]string{
						"team": team,
					},
				},
			},
			map[string]string{
				"app":  app,
				"team": team,
			},
		},
		{
			"test object meta custom label",
			&nais_io_v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels: map[string]string{
						"team": team,
						key:    value,
					},
				},
			},
			map[string]string{
				"app":  app,
				"team": team,
				key:    value,
			},
		},
		{
			"test object meta app label not overrideable",
			&nais_io_v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      app,
					Namespace: namespace,
					Labels: map[string]string{
						"team": team,
						"app":  "ignored",
					},
				},
			},
			map[string]string{
				"app":  app,
				"team": team,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateObjectMeta(tt.in)
			if !reflect.DeepEqual(got.Labels, tt.want) {
				t.Errorf("CreateObjectMeta().Labels = %v, want %v", got.Labels, tt.want)
			}
		})
	}
}
