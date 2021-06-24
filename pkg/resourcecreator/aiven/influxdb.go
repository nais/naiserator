package aiven

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Influx(ast *resource.Ast, influx *nais_io_v1.Influx) {
	if influx == nil {
		return
	}

	ast.Labels["aiven"] = "enabled"
}
