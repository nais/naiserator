package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

func Influx(ast *resource.Ast, influx *nais_io_v1.Influx, aivenApp *aiven_nais_io_v1.AivenApplication, credentialsEnabled bool) (bool, error) {
	if influx == nil {
		return false, nil
	}

	if credentialsEnabled {
		if influx.Instance == "" {
			return false, fmt.Errorf("Influx enabled, but no instance specified")
		}

		addInfluxEnvVariables(ast, aivenApp.Spec.SecretName)
		aivenApp.Spec.InfluxDB = &aiven_nais_io_v1.InfluxDBSpec{
			Instance: influx.Instance,
		}
	}

	ast.Labels["aiven"] = "enabled"

	return credentialsEnabled, nil
}

func addInfluxEnvVariables(ast *resource.Ast, secretName string) {
	// Add environment variables for string data
	ast.PrependEnv([]corev1.EnvVar{
		makeSecretEnvVar("INFLUXDB_USERNAME", secretName),
		makeSecretEnvVar("INFLUXDB_PASSWORD", secretName),
		makeSecretEnvVar("INFLUXDB_URI", secretName),
		makeOptionalSecretEnvVar("INFLUXDB_HOST", secretName),
		makeOptionalSecretEnvVar("INFLUXDB_PORT", secretName),
		makeSecretEnvVar("INFLUXDB_NAME", secretName),
	}...)
}
