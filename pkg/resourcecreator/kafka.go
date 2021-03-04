package resourcecreator

import (
	"fmt"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/util"
)

func generateKafkaSecretName(app *nais.Application) (string, error) {
	return util.GenerateSecretName("kafka", fmt.Sprintf("%s-%s", app.Name, app.Spec.Kafka.Pool), MaxSecretNameLength)
}
