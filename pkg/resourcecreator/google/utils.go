package google

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
)

func GcpServiceAccountName(app *nais_io_v1alpha1.Application, projectId string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", app.CreateAppNamespaceHash(), projectId)
}
