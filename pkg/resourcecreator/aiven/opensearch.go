package aiven

import (
	"fmt"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

func OpenSearch(ast *resource.Ast, openSearch *nais_io_v1.OpenSearch, aivenApp *aiven_nais_io_v1.AivenApplication) (bool, error) {
	if openSearch == nil {
		return false, nil
	}

	if openSearch.Instance == "" {
		return false, fmt.Errorf("OpenSearch enabled, but no instance specified")
	}

	aivenApp.Spec.OpenSearch = &aiven_nais_io_v1.OpenSearchSpec{
		Instance: fmt.Sprintf("opensearch-%s-%s", aivenApp.GetNamespace(), openSearch.Instance),
		Access:   openSearch.Access,
	}
	ast.Labels["aiven"] = "enabled"

	return true, nil
}
