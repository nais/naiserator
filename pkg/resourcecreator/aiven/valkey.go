package aiven

import (
	"fmt"

	aiven_io_v1alpha1 "github.com/nais/liberator/pkg/apis/aiven.io/v1alpha1"
	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		addValkeyEnvVariables(ast, aivenApp.Spec.SecretName, valkey.Instance)
		// Make the transition easier for teams coming from Redis by setting the `REDIS_` env variables too
		addRedisEnvVariables(ast, aivenApp.Spec.SecretName, valkey.Instance)

		aivenApp.Spec.Valkey = append(aivenApp.Spec.Valkey, &aiven_nais_io_v1.ValkeySpec{
			Instance: valkey.Instance,
			Access:   valkey.Access,
		})

		addDefaultValkeyIfNotExists(ast, source, config.GetAivenProject(), valkey.Instance)
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addDefaultValkeyIfNotExists(ast *resource.Ast, source Source, aivenProject, instanceName string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("valkey-%s-%s", source.GetNamespace(), instanceName)

	aivenValkey := &aiven_io_v1alpha1.Valkey{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Valkey",
			APIVersion: "aiven.io/v1alpha1",
		},
		ObjectMeta: objectMeta,
		Spec: aiven_io_v1alpha1.ValkeySpec{
			ServiceCommonSpec: aiven_io_v1alpha1.ServiceCommonSpec{
				Project: aivenProject,
				Plan:    "startup-4",
				Tags: map[string]string{
					"app": source.GetName(),
				},
			},
		},
	}
	ast.AppendOperation(resource.OperationCreateIfNotExists, aivenValkey)
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
