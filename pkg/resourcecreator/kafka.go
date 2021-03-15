package resourcecreator

import (
	"fmt"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
)

func generateKafkaSecretName(app *nais.Application) (string, error) {
	secretName, err := namegen.ShortName(fmt.Sprintf("kafka-%s-%s", app.Name, app.Spec.Kafka.Pool), MaxSecretNameLength)

	if err != nil {
		return "", fmt.Errorf("unable to generate kafka secret name: %s", err)
	}
	return secretName, err
}
