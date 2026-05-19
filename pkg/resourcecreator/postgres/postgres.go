package postgres

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/postgres/cnpg"
	"github.com/nais/naiserator/pkg/resourcecreator/postgres/zalando"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	// ==== These constants are copied from pgrator ====

	// ActiveEngineAnnotation is set by the operator to persist the engine choice
	// after first reconcile. Used to detect and reject engine changes.
	ActiveEngineAnnotation = "postgres.nais.io/active-engine"

	// EngineZalando is the Zalando Postgres Operator engine (default).
	EngineZalando = "zalando"
	// EngineCNPG is the CloudNativePG engine.
	EngineCNPG = "cnpg"
)

type Source interface {
	resource.Source
	GetPostgres() *nais_io_v1.Postgres
}

type Config interface {
	GetPostgresClusterEngine() string
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	postgres := source.GetPostgres()
	if postgres == nil {
		return nil
	}

	engine := cfg.GetPostgresClusterEngine()
	if engine == EngineZalando {
		zalando.Create(source, ast, postgres)
	} else if engine == EngineCNPG {
		cnpg.Create(source, ast, postgres)
	} else {
		return fmt.Errorf("unknown postgres engine: %v", engine)
	}

	return nil
}
