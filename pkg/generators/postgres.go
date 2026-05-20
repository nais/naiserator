package generators

import (
	"context"
	"fmt"
	"slices"

	"github.com/nais/naiserator/pkg/resourcecreator/postgres"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func preparePostgres(ctx context.Context, source postgres.Source, kube client.Client, o *Options) error {
	if source.GetPostgres() == nil {
		return nil
	}

	key := client.ObjectKey{
		Name:      source.GetPostgres().ClusterName,
		Namespace: source.GetNamespace(),
	}

	postgresMeta := &v1.PartialObjectMetadata{
		TypeMeta: v1.TypeMeta{
			Kind:       "Postgres",
			APIVersion: "data.nais.io/v1",
		},
	}
	err := kube.Get(ctx, key, postgresMeta)
	if err != nil {
		return fmt.Errorf("failed to get postgres cluster: %w", err)
	}

	engine := postgresMeta.GetAnnotations()[postgres.ActiveEngineAnnotation]
	if engine == "" {
		engine = postgresMeta.GetAnnotations()[postgres.EngineAnnotation]
	}
	if engine == "" {
		engine = postgres.EngineZalando
	}
	if !slices.Contains(postgres.AllEngines, engine) {
		return fmt.Errorf("unknown postgres engine: %v", engine)
	}

	o.PostgresClusterEngine = engine
	return nil
}
