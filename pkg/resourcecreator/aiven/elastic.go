package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

func Elastic(ast *resource.Ast, elastic *nais_io_v1.Elastic, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	if elastic == nil {
		return false, nil
	}

	if elastic.Instance == "" {
		return false, fmt.Errorf("Elastic enabled, but no instance specified")
	}

	addElasticEnvVariables(ast, aivenApp.Spec.SecretName)
	aivenApp.Spec.Elastic = &aiven_nais_io_v1.ElasticSpec{
		Instance: fmt.Sprintf("elastic-%s-%s", aivenApp.GetNamespace(), elastic.Instance),
		Access:   elastic.Access,
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addElasticEnvVariables(ast *resource.Ast, secretName string) {
	// Add environment variables for string data
	ast.Env = append(ast.Env, []corev1.EnvVar{
		makeSecretEnvVar("ELASTIC_USERNAME", secretName),
		makeSecretEnvVar("ELASTIC_PASSWORD", secretName),
		makeSecretEnvVar("ELASTIC_URI", secretName),
	}...)
}
