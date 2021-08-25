package aiven

import (
	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Influx(ast *resource.Ast, influx *nais_io_v1.Influx, aivenApp *aiven_nais_io_v1.AivenApplication) {
	if influx == nil {
		return
	}

	ast.Labels["aiven"] = "enabled"
}
