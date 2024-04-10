package aiven

import (
	"fmt"

	aiven_io_v1alpha1 "github.com/nais/liberator/pkg/apis/aiven.io/v1alpha1"
	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func OpenSearch(ast *resource.Ast, config Config, source Source, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	openSearch := source.GetOpenSearch()
	if openSearch == nil {
		return false, nil
	}

	if len(config.GetAivenProject()) == 0 {
		return false, fmt.Errorf("aiven project not defined for this cluster; needed for OpenSearch")
	}

	if openSearch.Instance == "" {
		return false, fmt.Errorf("OpenSearch requires instance name")
	}

	addOpenSearchEnvVariables(ast, aivenApp.Spec.SecretName)
	aivenApp.Spec.OpenSearch = &aiven_nais_io_v1.OpenSearchSpec{
		Instance: fmt.Sprintf("opensearch-%s-%s", aivenApp.GetNamespace(), openSearch.Instance),
		Access:   openSearch.Access,
	}
	addDefaultOpenSearchIfNotExists(ast, source, config.GetAivenProject(), openSearch.Instance)
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addDefaultOpenSearchIfNotExists(ast *resource.Ast, source Source, aivenProject, instanceName string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("opensearch-%s-%s", source.GetNamespace(), instanceName)

	aivenRedis := &aiven_io_v1alpha1.OpenSearch{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenSearch",
			APIVersion: "aiven.io/v1alpha",
		},
		ObjectMeta: objectMeta,
		Spec: aiven_io_v1alpha1.OpenSearchSpec{
			ServiceCommonSpec: aiven_io_v1alpha1.ServiceCommonSpec{
				Project: aivenProject,
				Plan:    DefaultPlan,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, aivenRedis)
}

func addOpenSearchEnvVariables(ast *resource.Ast, secretName string) {
	// Add environment variables for string data
	ast.Env = append(ast.Env, []corev1.EnvVar{
		makeSecretEnvVar("OPEN_SEARCH_USERNAME", secretName),
		makeSecretEnvVar("OPEN_SEARCH_PASSWORD", secretName),
		makeSecretEnvVar("OPEN_SEARCH_URI", secretName),
	}...)
}
