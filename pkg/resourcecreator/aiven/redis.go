package aiven

import (
	"fmt"
	"regexp"
	"strings"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

var namePattern = regexp.MustCompile("[^a-z0-9]")

func Redis(ast *resource.Ast, redises []nais_io_v1.Redis, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	if len(redises) == 0 {
		return false, nil
	}

	for _, redis := range redises {
		if redis.Instance == "" {
			return false, fmt.Errorf("Redis requires instance name")
		}

		addRedisEnvVariables(ast, aivenApp.Spec.SecretName, redis.Instance)
		aivenApp.Spec.Redis = append(aivenApp.Spec.Redis, &aiven_nais_io_v1.RedisSpec{
			Instance: redis.Instance,
			Access:   redis.Access,
		})
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addRedisEnvVariables(ast *resource.Ast, secretName, instanceName string) {
	// Add environment variables for string data
	suffix := envVarSuffix(instanceName)
	ast.Env = append(ast.Env, []corev1.EnvVar{
		makeSecretEnvVar(fmt.Sprintf("REDIS_USERNAME_%s", suffix), secretName),
		makeSecretEnvVar(fmt.Sprintf("REDIS_PASSWORD_%s", suffix), secretName),
		makeSecretEnvVar(fmt.Sprintf("REDIS_URI_%s", suffix), secretName),
	}...)
}

func envVarSuffix(instanceName string) string {
	return strings.ToUpper(namePattern.ReplaceAllString(instanceName, "_"))
}
