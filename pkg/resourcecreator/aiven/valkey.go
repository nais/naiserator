package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

func Valkey(ast *resource.Ast, config Config, source Source, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	valkeyes := source.GetValkey()
	if len(valkeyes) == 0 {
		return false, nil
	}

	if len(config.GetAivenProject()) == 0 {
		return false, fmt.Errorf("aiven project not defined for this cluster; needed for Valkey")
	}

	for _, valkey := range valkeyes {
		if valkey.Instance == "" {
			return false, fmt.Errorf("Valkey requires instance name")
		}

		secretName, err := generateAivenSecretName(aivenApp.Name, fmt.Sprintf("valkey-%s", valkey.Instance), aivenApp.ObjectMeta.Labels["aiven.nais.io/secret-generation"])
		if err != nil {
			return false, err
		}

		addValkeyEnvVariables(ast, secretName, valkey.Instance)
		// Make the transition easier for teams coming from Redis by setting the `REDIS_` env variables too
		addRedisEnvVariables(ast, secretName, valkey.Instance)
		ast.PrependEnv([]corev1.EnvVar{makeOptionalSecretEnvVar("AIVEN_CA", secretName)}...)

		aivenApp.Spec.Valkey = append(aivenApp.Spec.Valkey, &aiven_nais_io_v1.ValkeySpec{
			Instance:   valkey.Instance,
			Access:     valkey.Access,
			SecretName: secretName,
		})
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addValkeyEnvVariables(ast *resource.Ast, secretName, instanceName string) {
	// Add environment variables for string data
	suffix := envVarSuffix(instanceName)
	ast.PrependEnv([]corev1.EnvVar{
		makeSecretEnvVar(fmt.Sprintf("VALKEY_USERNAME_%s", suffix), secretName),
		makeSecretEnvVar(fmt.Sprintf("VALKEY_PASSWORD_%s", suffix), secretName),
		makeSecretEnvVar(fmt.Sprintf("VALKEY_URI_%s", suffix), secretName),
		makeOptionalSecretEnvVar(fmt.Sprintf("VALKEY_HOST_%s", suffix), secretName),
		makeOptionalSecretEnvVar(fmt.Sprintf("VALKEY_PORT_%s", suffix), secretName),
	}...)
}

func addRedisEnvVariables(ast *resource.Ast, secretName, instanceName string) {
	// Add environment variables for string data
	suffix := envVarSuffix(instanceName)
	ast.PrependEnv([]corev1.EnvVar{
		makeSecretEnvVar(fmt.Sprintf("REDIS_USERNAME_%s", suffix), secretName),
		makeSecretEnvVar(fmt.Sprintf("REDIS_PASSWORD_%s", suffix), secretName),
		makeSecretEnvVar(fmt.Sprintf("REDIS_URI_%s", suffix), secretName),
		makeOptionalSecretEnvVar(fmt.Sprintf("REDIS_HOST_%s", suffix), secretName),
		makeOptionalSecretEnvVar(fmt.Sprintf("REDIS_PORT_%s", suffix), secretName),
	}...)
}
