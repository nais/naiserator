package aiven

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Elastic(ast *resource.Ast, elastic *nais_io_v1alpha1.Elastic) {
	if elastic == nil {
		return
	}

	ast.Labels["aiven"] = "enabled"
}
