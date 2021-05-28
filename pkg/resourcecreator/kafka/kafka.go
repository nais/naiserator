package kafka

import (
	"fmt"
	"path/filepath"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func podSpecWithVolume(spec *corev1.PodSpec, volume corev1.Volume) *corev1.PodSpec {
	spec.Volumes = append(spec.Volumes, volume)
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, pod.FromFilesVolumeMount(volume.Name, nais_io_v1alpha1.DefaultKafkaratorMountPath, ""))
	return spec
}

func generateKafkaSecretName(name, pool string) (string, error) {
	secretName, err := namegen.ShortName(fmt.Sprintf("kafka-%s-%s", name, pool), validation.DNS1035LabelMaxLength)

	if err != nil {
		return "", fmt.Errorf("unable to generate kafka secret name: %s", err)
	}
	return secretName, err
}
func podSpecWithKafka(podSpec *corev1.PodSpec, kafkaratorSecretName string) *corev1.PodSpec {
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
	podSpec = podSpecWithVolume(podSpec, credentialFilesVolume)

	// Add environment variables for string data
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, []corev1.EnvVar{
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
	podSpec.Containers[0].Env = append(podSpec.Containers[0].Env, []corev1.EnvVar{
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

	return podSpec
}

func Create(objectMeta metav1.ObjectMeta, resourceOptions resource.Options, deployment *appsv1.Deployment, naisKafka *nais_io_v1alpha1.Kafka) error {
	if resourceOptions.KafkaratorEnabled && naisKafka != nil {
		kafkaratorSecretName, err := generateKafkaSecretName(objectMeta.Name, naisKafka.Pool)
		if err != nil {
			return err
		}

		podSpec := &deployment.Spec.Template.Spec
		podSpec = podSpecWithKafka(podSpec, kafkaratorSecretName)
		deployment.Spec.Template.Spec = *podSpec
		deployment.Spec.Template.ObjectMeta.Labels["kafka"] = "enabled"
	}
	return nil
}
