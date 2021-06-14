package kafka

import (
	"fmt"
	"path/filepath"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	kafkaCredentialFilesVolumeName = "kafka-credentials"
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

func makeKafkaSecretEnvVar(key, secretName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: key,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}

func generateKafkaSecretName(name, pool string) string {
	secretName := namegen.RandShortName(fmt.Sprintf("kafka-%s-%s", name, pool), validation.DNS1035LabelMaxLength)

	return secretName
}

func podSpecWithKafka(ast *resource.Ast, kafkaratorSecretName string) {
	// Mount specific secret keys as credential files
	credentialFilesVolume := pod.FromFilesSecretVolume(kafkaCredentialFilesVolumeName, kafkaratorSecretName, []corev1.KeyToPath{
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
	})
	ast.Volumes = append(ast.Volumes, credentialFilesVolume)
	ast.VolumeMounts = append(ast.VolumeMounts, pod.FromFilesVolumeMount(credentialFilesVolume.Name, nais_io_v1alpha1.DefaultKafkaratorMountPath, ""))

	// Add environment variables for string data
	ast.Env = append(ast.Env, []corev1.EnvVar{
		makeKafkaSecretEnvVar(kafkaCertificateKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaPrivateKeyKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaBrokersKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaSchemaRegistryKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaSchemaRegistryUserKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaSchemaRegistryPasswordKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaCAKey, kafkaratorSecretName),
		makeKafkaSecretEnvVar(kafkaCredStorePasswordKey, kafkaratorSecretName),
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

func Create(source resource.Source, ast *resource.Ast, resourceOptions resource.Options, naisKafka *nais_io_v1.Kafka) {
	if resourceOptions.KafkaratorEnabled && naisKafka != nil {
		secretName := generateKafkaSecretName(source.GetName(), naisKafka.Pool)

		podSpecWithKafka(ast, secretName)
		ast.Labels["kafka"] = "enabled"
		aivenApp := aiven_nais_io_v1.NewAivenApplicationBuilder(source.GetName(), source.GetNamespace()).
			WithSpec(aiven_nais_io_v1.AivenApplicationSpec{
				SecretName: secretName,
				Kafka: aiven_nais_io_v1.KafkaSpec{
					Pool: naisKafka.Pool,
				},
			}).
			Build()
		ast.AppendOperation(resource.OperationCreateOrUpdate, &aivenApp)
	}
}
