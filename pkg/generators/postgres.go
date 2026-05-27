package generators

import (
	"context"
	"fmt"
	"slices"

	"github.com/nais/naiserator/pkg/resourcecreator/postgres"
	"github.com/nais/pgrator/pkg/api/datav1"
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

	pg := &datav1.Postgres{}
	err := kube.Get(ctx, key, pg)
	if err != nil {
		return fmt.Errorf("failed to get postgres cluster: %w", err)
	}

	var engine string
	if pg.Status != nil {
		engine = pg.Status.Engine
	}
	if engine == "" {
		return fmt.Errorf("waiting for pgrator to set engine in status on %s/%s; will retry", pg.GetNamespace(), pg.GetName())
	}
	if !slices.Contains(postgres.AllEngines, engine) {
		return fmt.Errorf("unknown postgres engine: %v", engine)
	}

	o.PostgresClusterEngine = engine
	return nil
}
