package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const ApplicationName = "myapplication"
const ApplicationNamespace = "mynamespace"
const ApplicationTeam = "myteam"

type realObjects struct {
	deployment *v1.Deployment
}

func getRealObjects(resources []runtime.Object) realObjects {
	real := realObjects{}
	for _, r := range resources {
		switch v := r.(type) {
		case *v1.Deployment:
			real.deployment = v
		}
	}
	return real
}

func minimalFailingApplication() *nais.Application {
	return &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
		},
	}
}

func minimalApplication() *nais.Application {
	return &nais.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApplicationName,
			Namespace: ApplicationNamespace,
			Labels: map[string]string{
				"team": ApplicationTeam,
			},
		},
	}
}

// Test that a specified application spec results in the correct Kubernetes resources.
func TestCreate(t *testing.T) {
	t.Run("default application spec merges into empty struct", func(t *testing.T) {
		app := &nais.Application{}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
	})

	t.Run("application spec needs required parameters", func(t *testing.T) {
		app := minimalFailingApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("team label and application name is propagated to created resources", func(t *testing.T) {
		app := minimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Equal(t, app.Name, objects.deployment.Name)
		assert.Equal(t, app.Namespace, objects.deployment.Namespace)
		assert.Equal(t, app.Name, objects.deployment.Labels["app"])
		assert.Equal(t, app.Labels["team"], objects.deployment.Labels["team"])

	})
}
