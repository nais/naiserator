package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	rbac_istio_io_v1alpha1 "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const ApplicationName = "myapplication"
const ApplicationNamespace = "mynamespace"
const ApplicationTeam = "myteam"

type realObjects struct {
	deployment         *v1.Deployment
	service            *corev1.Service
	serviceAccount     *corev1.ServiceAccount
	hpa                *autoscalingv1.HorizontalPodAutoscaler
	ingress            *extensionsv1beta1.Ingress
	networkPolicy      *networkingv1.NetworkPolicy
	serviceRole        *rbac_istio_io_v1alpha1.ServiceRole
	serviceRoleBinding *rbac_istio_io_v1alpha1.ServiceRoleBinding
	virtualService     *networking_istio_io_v1alpha3.VirtualService
}

func getRealObjects(resources []runtime.Object) realObjects {
	real := realObjects{}
	for _, r := range resources {
		switch v := r.(type) {
		case *v1.Deployment:
			real.deployment = v
		case *corev1.Service:
			real.service = v
		case *corev1.ServiceAccount:
			real.serviceAccount = v
		case *autoscalingv1.HorizontalPodAutoscaler:
			real.hpa = v
		case *extensionsv1beta1.Ingress:
			real.ingress = v
		case *networkingv1.NetworkPolicy:
			real.networkPolicy = v
		case *rbac_istio_io_v1alpha1.ServiceRole:
			real.serviceRole = v
		case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
			real.serviceRoleBinding = v
		case *networking_istio_io_v1alpha3.VirtualService:
			real.virtualService = v
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

	t.Run("all basic resource types are created from an application spec", func(t *testing.T) {
		app := minimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.hpa)
		assert.NotNil(t, objects.service)
		assert.NotNil(t, objects.serviceAccount)
		assert.NotNil(t, objects.deployment)
		assert.Nil(t, objects.ingress)
	})

	t.Run("an ingress object is created if ingress paths are specified", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.Ingresses = []string{"https://foo.bar/baz"}
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.ingress)
	})

	t.Run("erroneous ingress uris create errors", func(t *testing.T) {
		app := minimalApplication()
		app.Spec.Ingresses = []string{"gopher://lol"}
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("istio resources are omitted when access policy creation is disabled", func(t *testing.T) {
		app := minimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Nil(t, objects.virtualService)
		assert.Nil(t, objects.serviceRoleBinding)
		assert.Nil(t, objects.serviceRole)
		assert.Nil(t, objects.networkPolicy)
	})

	t.Run("istio resources are created when access policy creation is enabled", func(t *testing.T) {
		app := minimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.virtualService)
		assert.NotNil(t, objects.serviceRoleBinding)
		assert.NotNil(t, objects.serviceRole)
		assert.NotNil(t, objects.networkPolicy)
	})
}
