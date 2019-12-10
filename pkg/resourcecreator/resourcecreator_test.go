package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	rbac_istio_io_v1alpha1 "github.com/nais/naiserator/pkg/apis/rbac.istio.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type realObjects struct {
	deployment          *v1.Deployment
	service             *corev1.Service
	serviceAccount      *corev1.ServiceAccount
	hpa                 *autoscalingv1.HorizontalPodAutoscaler
	ingress             *extensionsv1beta1.Ingress
	networkPolicy       *networkingv1.NetworkPolicy
	serviceRoles        []*rbac_istio_io_v1alpha1.ServiceRole
	serviceRoleBindings []*rbac_istio_io_v1alpha1.ServiceRoleBinding
	virtualServices     []*networking_istio_io_v1alpha3.VirtualService
	role                *rbacv1.Role
	rolebinding         *rbacv1.RoleBinding
}

func getRealObjects(resources resourcecreator.ResourceOperations) (o realObjects) {
	for _, r := range resources {
		switch v := r.Resource.(type) {
		case *v1.Deployment:
			o.deployment = v
		case *corev1.Service:
			o.service = v
		case *corev1.ServiceAccount:
			o.serviceAccount = v
		case *autoscalingv1.HorizontalPodAutoscaler:
			o.hpa = v
		case *extensionsv1beta1.Ingress:
			o.ingress = v
		case *networkingv1.NetworkPolicy:
			o.networkPolicy = v
		case *rbac_istio_io_v1alpha1.ServiceRole:
			o.serviceRoles = append(o.serviceRoles, v)
		case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
			o.serviceRoleBindings = append(o.serviceRoleBindings, v)
		case *networking_istio_io_v1alpha3.VirtualService:
			o.virtualServices = append(o.virtualServices, v)
		case *rbacv1.Role:
			o.role = v
		case *rbacv1.RoleBinding:
			o.rolebinding = v
		}
	}
	return
}

// Test that a specified application spec results in the correct Kubernetes resources.
func TestCreate(t *testing.T) {
	t.Run("default application spec merges into empty struct", func(t *testing.T) {
		app := &nais.Application{}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
	})

	t.Run("application spec needs required parameters", func(t *testing.T) {
		app := fixtures.MinimalFailingApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("team label and application name is propagated to created resources", func(t *testing.T) {
		app := fixtures.MinimalApplication()
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
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources.Extract(resourcecreator.OperationCreateOrUpdate))
		assert.NotNil(t, objects.hpa)
		assert.NotNil(t, objects.service)
		assert.NotNil(t, objects.serviceAccount)
		assert.NotNil(t, objects.deployment)
		assert.Nil(t, objects.ingress)

		// Test that the Ingress is deleted
		objects = getRealObjects(resources.Extract(resourcecreator.OperationDeleteIfExists))
		assert.NotNil(t, objects.ingress)
	})

	t.Run("an ingress object is created if ingress paths are specified", func(t *testing.T) {
		app := fixtures.MinimalApplication()
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
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"gopher://lol"}
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("istio resources are omitted when access policy creation is disabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Nil(t, objects.virtualServices)
		assert.Nil(t, objects.serviceRoleBindings)
		assert.Nil(t, objects.serviceRoles)
		assert.Nil(t, objects.networkPolicy)
	})

	t.Run("istio resources are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []string{"https://host.domain.tld"}
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Len(t, objects.virtualServices, 1)
		assert.NotNil(t, objects.serviceRoleBindings)
		assert.NotNil(t, objects.serviceRoles)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("servicerole and servicerolebinding resources are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		app.Spec.Ingresses = []string{"https://foo.bar"}

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.virtualServices)
		assert.NotNil(t, objects.serviceRoleBindings)
		assert.NotNil(t, objects.serviceRoles)
		assert.NotNil(t, objects.networkPolicy)
	})

	t.Run("servicerolebinding and prometheus servicerolebinding resources are created when access policy creation is enabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		app.Spec.AccessPolicy.Inbound.Rules = []nais.AccessPolicyRule{{"otherapp", "othernamespace"}}
		app.Spec.Prometheus.Enabled = true

		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.Len(t, objects.serviceRoles, 2)
		assert.Len(t, objects.serviceRoleBindings, 2)
	})

	t.Run("leader election rbac is created when LE is requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.role)
		assert.Equal(t, app.Name, objects.role.Name)
		assert.NotNil(t, objects.rolebinding)
		assert.Equal(t, app.Name, objects.rolebinding.Name)
	})

	t.Run("default network policy that allows egress to resources in kube-system and istio-system is created for app", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true
		opts.AccessPolicyNotAllowedCIDRs = []string{"101.0.0.0/8"}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		objects := getRealObjects(resources)
		assert.NotNil(t, objects.networkPolicy)

		assert.NotNil(t, objects.networkPolicy)
		assert.NotEmpty(t, objects.networkPolicy.Spec.Egress)
	})

	t.Run("omitting ingresses denies traffic from istio ingress gateway", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		opts.AccessPolicy = true

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		deletes := resources.Extract(resourcecreator.OperationDeleteIfExists)
		numDeletes := 0
		for _, resource := range deletes {
			switch x := resource.Resource.(type) {
			case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
				if x.GetName() == "myapplication" {
					numDeletes++
				}
			}
		}

		if numDeletes != 1 {
			t.Fail()
		}
	})

	t.Run("no service role and no service role binding created for prometheus, when disabled", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()

		app.Spec.Prometheus.Enabled = false
		opts.AccessPolicy = true

		resources, err := resourcecreator.Create(app, opts)
		assert.NoError(t, err)

		deletes := resources.Extract(resourcecreator.OperationDeleteIfExists)
		numDeletes := 0
		for _, resource := range deletes {
			switch x := resource.Resource.(type) {
			case *rbac_istio_io_v1alpha1.ServiceRoleBinding:
				if x.GetName() == "myapplication-prometheus" {
					numDeletes++
				}
			}
		}

		if numDeletes != 1 {
			t.Fail()
		}
	})

}
