package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Elastic(source resource.Source, ast *resource.Ast, elastic *nais_io_v1.Elastic, aivenApp *aiven_nais_io_v1.AivenApplication) error {
	if elastic == nil {
		return nil
	}

	if elastic.Instance == "" {
		return fmt.Errorf("Elastic enabled, but no instance specified")
	}

	ast.Labels["aiven"] = "enabled"
	return nil
}
