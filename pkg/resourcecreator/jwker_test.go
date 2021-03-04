package resourcecreator_test

import (
	"testing"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestJwker(t *testing.T) {
	otherApplication := "a"
	clusterName := "myCluster"
	otherNamespace := "othernamespace"
	otherCluster := "otherCluster"
	otherApplication2 := "b"
	otherNamespace2 := "othernamespace2"
	otherApplication3 := "c"

	fixture := func() *nais_io_v1alpha1.Application {
		app := fixtures.MinimalApplication()
		app.Spec.TokenX = &nais_io_v1alpha1.TokenX{
			Enabled: true,
		}
		return app
	}

	t.Run("jwker for app with no access policy", func(t *testing.T) {
		app := fixture()
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.NotEmpty(t, jwker)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 0)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 0)
	})

	t.Run("one inbound without cluster/namespace and no outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, "", ""}}
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 1)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jwker.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 0)
	})

	t.Run("one inbound with cluster/namespace and no outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, otherCluster}}
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 1)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jwker.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jwker.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 0)
	})

	t.Run("one outbound and no inbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Outbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, otherCluster}}
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 1)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Outbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jwker.Spec.AccessPolicy.Outbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jwker.Spec.AccessPolicy.Outbound.Rules[0].Cluster)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 0)
	})

	t.Run("multiple inbound and no outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 3)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 0)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jwker.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jwker.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jwker.Spec.AccessPolicy.Inbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jwker.Spec.AccessPolicy.Inbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Inbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jwker.Spec.AccessPolicy.Inbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jwker.Spec.AccessPolicy.Inbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Inbound.Rules[2].Cluster)
	})

	t.Run("multiple outbound and no inbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Outbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 3)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 0)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Outbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jwker.Spec.AccessPolicy.Outbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jwker.Spec.AccessPolicy.Outbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jwker.Spec.AccessPolicy.Outbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jwker.Spec.AccessPolicy.Outbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Outbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jwker.Spec.AccessPolicy.Outbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jwker.Spec.AccessPolicy.Outbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Outbound.Rules[2].Cluster)
	})
	//
	t.Run("multiple inbound and multiple outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}

		app.Spec.AccessPolicy.Outbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}
		jwker, err := resourcecreator.Jwker(app, clusterName)
		assert.NoError(t, err)
		assert.Len(t, jwker.Spec.AccessPolicy.Inbound.Rules, 3)
		assert.Len(t, jwker.Spec.AccessPolicy.Outbound.Rules, 3)
		assert.NotEmpty(t, jwker.Spec.SecretName)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jwker.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jwker.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jwker.Spec.AccessPolicy.Inbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jwker.Spec.AccessPolicy.Inbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Inbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jwker.Spec.AccessPolicy.Inbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jwker.Spec.AccessPolicy.Inbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Inbound.Rules[2].Cluster)
		assert.Equal(t, otherApplication, jwker.Spec.AccessPolicy.Outbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jwker.Spec.AccessPolicy.Outbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jwker.Spec.AccessPolicy.Outbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jwker.Spec.AccessPolicy.Outbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jwker.Spec.AccessPolicy.Outbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Outbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jwker.Spec.AccessPolicy.Outbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jwker.Spec.AccessPolicy.Outbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jwker.Spec.AccessPolicy.Outbound.Rules[2].Cluster)
	})
}
