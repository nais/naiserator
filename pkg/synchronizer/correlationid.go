package synchronizer

import (
	"fmt"

	"github.com/google/uuid"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

// If the application was not deployed with a correlation ID annotation,
// generate a random UUID and add it to annotations.
func ensureCorrelationID(source resource.Source) error {
	anno := source.GetAnnotations()
	if anno == nil {
		anno = make(map[string]string)
	}

	if len(anno[nais_io_v1.DeploymentCorrelationIDAnnotation]) != 0 {
		return nil
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("generate deployment correlation ID: %s", err)
	}

	anno[nais_io_v1.DeploymentCorrelationIDAnnotation] = id.String()

	source.SetAnnotations(anno)

	return nil
}
