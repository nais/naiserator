package aiven

import (
	"path/filepath"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	kafka_nais_io_v1 "github.com/nais/liberator/pkg/apis/kafka.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	kafkaCertificatePathKey        = "KAFKA_CERTIFICATE_PATH"
	kafkaPrivateKeyPathKey         = "KAFKA_PRIVATE_KEY_PATH"
	kafkaCAPathKey                 = "KAFKA_CA_PATH"
	kafkaKeystorePathKey           = "KAFKA_KEYSTORE_PATH"
	kafkaTruststorePathKey         = "KAFKA_TRUSTSTORE_PATH"
	kafkaCertificateKey            = "KAFKA_CERTIFICATE"
	kafkaPrivateKeyKey             = "KAFKA_PRIVATE_KEY"
	kafkaCAKey                     = "KAFKA_CA"
	kafkaBrokersKey                = "KAFKA_BROKERS"
	kafkaSchemaRegistryKey         = "KAFKA_SCHEMA_REGISTRY"
	kafkaSchemaRegistryUserKey     = "KAFKA_SCHEMA_REGISTRY_USER"
	kafkaSchemaRegistryPasswordKey = "KAFKA_SCHEMA_REGISTRY_PASSWORD"
	kafkaCredStorePasswordKey      = "KAFKA_CREDSTORE_PASSWORD"
	kafkaCertificateFilename       = "kafka.crt"
	kafkaPrivateKeyFilename        = "kafka.key"
	kafkaCAFilename                = "ca.crt"
	kafkaKeystoreFilename          = "client.keystore.p12"
	kafkaTruststoreFilename        = "client.truststore.jks"
)

func addKafkaEnvVariables(ast *resource.Ast, secretName string) {
	// Add environment variables for string data
	ast.Env = append(ast.Env, []corev1.EnvVar{
		makeSecretEnvVar(kafkaCertificateKey, secretName),
		makeSecretEnvVar(kafkaPrivateKeyKey, secretName),
		makeSecretEnvVar(kafkaBrokersKey, secretName),
		makeSecretEnvVar(kafkaSchemaRegistryKey, secretName),
		makeSecretEnvVar(kafkaSchemaRegistryUserKey, secretName),
		makeSecretEnvVar(kafkaSchemaRegistryPasswordKey, secretName),
		makeSecretEnvVar(kafkaCAKey, secretName),
		makeSecretEnvVar(kafkaCredStorePasswordKey, secretName),
	}...)

	// Inject path environment variables to refer to mounted secrets
	ast.Env = append(ast.Env, []corev1.EnvVar{
		{
			Name:  kafkaCertificatePathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaCertificateFilename),
		},
		{
			Name:  kafkaPrivateKeyPathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaPrivateKeyFilename),
		},
		{
			Name:  kafkaCAPathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaCAFilename),
		},
		{
			Name:  kafkaKeystorePathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaKeystoreFilename),
		},
		{
			Name:  kafkaTruststorePathKey,
			Value: filepath.Join(nais_io_v1alpha1.DefaultKafkaratorMountPath, kafkaTruststoreFilename),
		},
	}...)
}

func createKafkaKeyToPaths() []corev1.KeyToPath {
	// Mount specific secret keys as credential files
	return []corev1.KeyToPath{
		{
			Key:  kafkaCertificateKey,
			Path: kafkaCertificateFilename,
		},
		{
			Key:  kafkaPrivateKeyKey,
			Path: kafkaPrivateKeyFilename,
		},
		{
			Key:  kafkaCAKey,
			Path: kafkaCAFilename,
		},
		{
			Key:  kafkaKeystoreFilename,
			Path: kafkaKeystoreFilename,
		},
		{
			Key:  kafkaTruststoreFilename,
			Path: kafkaTruststoreFilename,
		},
	}
}

func Kafka(source resource.Source, ast *resource.Ast, config Config, naisKafka *nais_io_v1.Kafka, aivenApp *aiven_nais_io_v1.AivenApplication) []corev1.KeyToPath {
	if config.IsKafkaratorEnabled() && naisKafka != nil {
		addKafkaEnvVariables(ast, aivenApp.Spec.SecretName)
		ast.Labels["kafka"] = "enabled"
		aivenApp.Spec.Kafka = &aiven_nais_io_v1.KafkaSpec{
			Pool: naisKafka.Pool,
		}

		if naisKafka.Streams {
			stream := CreateStream(source, naisKafka)
			ast.AppendOperation(resource.OperationCreateOrUpdate, stream)
			ast.Env = append(ast.Env, corev1.EnvVar{
				Name:  "KAFKA_STREAMS_APPLICATION_ID",
				Value: stream.TopicPrefix(),
			})
		}

		return createKafkaKeyToPaths()
	}

	return []corev1.KeyToPath{}
}

func CreateStream(source resource.Source, kafka *nais_io_v1.Kafka) *kafka_nais_io_v1.Stream {
	return &kafka_nais_io_v1.Stream{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Stream",
			APIVersion: "kafka.nais.io/v1",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec: kafka_nais_io_v1.StreamSpec{
			Pool: kafka.Pool,
		},
	}
}
