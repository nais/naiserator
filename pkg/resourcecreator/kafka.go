package resourcecreator

import (
	"fmt"
	"hash/crc32"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
)

const (
	maxSecretNameLength = 63
)

func generateKafkaSecretName(app *nais.Application) (string, error) {
	secretName, err := kafkaShortName(fmt.Sprintf("kafka-%s-%s", app.Name, app.Spec.Kafka.Pool), maxSecretNameLength)
	if err != nil {
		return "", fmt.Errorf("unable to generate kafka secret name: %s", err)
	}
	return secretName, err
}

// Copied from Kafkarator. Procedurally generate a short string with hash that can be calculated using the base name
func kafkaShortName(basename string, maxlen int) (string, error) {
	maxlen -= 9 // 8 bytes of hexadecimal hash and 1 byte of separator
	hasher := crc32.NewIEEE()
	_, err := hasher.Write([]byte(basename))
	if err != nil {
		return "", err
	}
	hashStr := fmt.Sprintf("%x", hasher.Sum32())
	if len(basename) > maxlen {
		basename = basename[:maxlen]
	}
	return basename + "-" + hashStr, nil
}
