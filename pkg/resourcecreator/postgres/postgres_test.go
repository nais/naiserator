package postgres_test

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/resourcecreator/postgres"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
)

// TestNilOwnerReferences ensures that the owner references are not set,
// since the postgres resources are created in a different namespace to the application.
// If we have an owner reference, it will point to a non-existing application in the pg namespace,
// and the postgres cluster will immediately be deleted by the garbage collector.
func TestNilOwnerReferences(t *testing.T) {
	source := fixtures.MinimalApplication()
	source.Spec.Postgres = &nais_io_v1.Postgres{
		Cluster: nais_io_v1.PostgresCluster{
			Resources: nais_io_v1.PostgresResources{
				DiskSize: k8sResource.MustParse("10Gi"),
				Cpu:      k8sResource.MustParse("1"),
				Memory:   k8sResource.MustParse("4G"),
			},
			MajorVersion: "17",
		},
	}

	ast := resource.NewAst()
	cfg := &generators.Options{}
	postgres.CreateClusterSpec(source, ast, cfg, "cluster-name", "pg-namespace")

	assert.Nil(t, ast.Operations[0].Resource.GetOwnerReferences())
}
