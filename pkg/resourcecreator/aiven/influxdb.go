package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func Influx(ast *resource.Ast, influx *nais_io_v1.Influx, aivenApp *aiven_nais_io_v1.AivenApplication, credentialsEnabled bool) (bool, error) {
	if influx == nil {
		return false, nil
	}

	if credentialsEnabled {
		if influx.Instance == "" {
			return false, fmt.Errorf("Influx enabled, but no instance specified")
		}

		aivenApp.Spec.InfluxDB = &aiven_nais_io_v1.InfluxDBSpec{
			Instance: influx.Instance,
		}
	}

	ast.Labels["aiven"] = "enabled"

	return credentialsEnabled, nil
}
