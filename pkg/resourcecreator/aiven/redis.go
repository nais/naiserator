package aiven

import (
	"fmt"
	"regexp"
	"strings"

	aiven_io_v1alpha1 "github.com/nais/liberator/pkg/apis/aiven.io/v1alpha1"
	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultPlan = "startup-4"

var namePattern = regexp.MustCompile("[^a-z0-9]")

func Redis(ast *resource.Ast, config Config, source Source, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	redises := source.GetRedis()
	if len(redises) == 0 {
		return false, nil
	}

	if len(config.GetAivenProject()) == 0 {
		return false, fmt.Errorf("NAISERATOR-6450: aiven project not defined for this cluster; needed for Redis")
	}

	for _, redis := range redises {
		if redis.Instance == "" {
			return false, fmt.Errorf("NAISERATOR-1564: Redis requires instance name")
		}

		addRedisEnvVariables(ast, aivenApp.Spec.SecretName, redis.Instance)
		aivenApp.Spec.Redis = append(aivenApp.Spec.Redis, &aiven_nais_io_v1.RedisSpec{
			Instance: redis.Instance,
			Access:   redis.Access,
		})

		addDefaultRedisIfNotExists(ast, source, config.GetAivenProject(), redis.Instance)
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addDefaultRedisIfNotExists(ast *resource.Ast, source Source, aivenProject, instanceName string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("redis-%s-%s", source.GetNamespace(), instanceName)

	aivenRedis := &aiven_io_v1alpha1.Redis{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Redis",
			APIVersion: "aiven.io/v1alpha1",
		},
		ObjectMeta: objectMeta,
		Spec: aiven_io_v1alpha1.RedisSpec{
			ServiceCommonSpec: aiven_io_v1alpha1.ServiceCommonSpec{
				Project: aivenProject,
				Plan:    DefaultPlan,
			},
		},
	}
	ast.AppendOperation(resource.OperationCreateIfNotExists, aivenRedis)
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
