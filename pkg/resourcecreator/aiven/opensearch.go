package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
)

func OpenSearch(ast *resource.Ast, openSearch *nais_io_v1.OpenSearch, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	if openSearch == nil {
		return false, nil
	}

	if openSearch.Instance == "" {
		return false, fmt.Errorf("OpenSearch enabled, but no instance specified")
	}

	secretName, err := generateAivenSecretName(aivenApp.Name, "opensearch", aivenApp.ObjectMeta.Labels["aiven.nais.io/secret-generation"])
	if err != nil {
		return false, err
	}

	ast.PrependEnv([]corev1.EnvVar{makeOptionalSecretEnvVar("AIVEN_CA", secretName)}...)
	addOpenSearchEnvVariables(ast, secretName)
	aivenApp.Spec.OpenSearch = &aiven_nais_io_v1.OpenSearchSpec{
		Instance:   fmt.Sprintf("opensearch-%s-%s", aivenApp.GetNamespace(), openSearch.Instance),
		Access:     openSearch.Access,
		SecretName: secretName,
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}

func addOpenSearchEnvVariables(ast *resource.Ast, secretName string) {
	// Add environment variables for string data
	ast.PrependEnv([]corev1.EnvVar{
		makeSecretEnvVar("OPEN_SEARCH_USERNAME", secretName),
		makeSecretEnvVar("OPEN_SEARCH_PASSWORD", secretName),
		makeSecretEnvVar("OPEN_SEARCH_URI", secretName),
		makeOptionalSecretEnvVar("OPEN_SEARCH_HOST", secretName),
		makeOptionalSecretEnvVar("OPEN_SEARCH_PORT", secretName),
	}...)
}
