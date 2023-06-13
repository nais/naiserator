package aiven

import (
	"fmt"
	"time"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/namegen"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
)

const (
	aivenCredentialFilesVolumeName = "aiven-credentials"
)

type Source interface {
	resource.Source
	GetInflux() *nais_io_v1.Influx
	GetKafka() *nais_io_v1.Kafka
	GetOpenSearch() *nais_io_v1.OpenSearch
	GetRedis() []nais_io_v1.Redis
}

type Config interface {
	IsKafkaratorEnabled() bool
	IsAivenSharedSecretsEnabled() bool
}

func generateAivenSecretName(name string) string {
	secretName := namegen.RandShortName(fmt.Sprintf("aiven-%s", name), validation.DNS1035LabelMaxLength)

	return secretName
}

func generateSharedAivenSecretName(name string) (string, error) {
	prefixedName := fmt.Sprintf("aiven-%s", name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d", year, week)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func Create(source Source, ast *resource.Ast, config Config) error {
	var secretName string
	if config.IsAivenSharedSecretsEnabled() {
		var err error
		secretName, err = generateSharedAivenSecretName(source.GetName())
		if err != nil {
			return err
		}
	} else {
		secretName = generateAivenSecretName(source.GetName())
	}

	aivenApp := aiven_nais_io_v1.NewAivenApplicationBuilder(source.GetName(), source.GetNamespace()).
		WithSpec(aiven_nais_io_v1.AivenApplicationSpec{
			SecretName: secretName,
		}).
		Build()
	aivenApp.ObjectMeta = resource.CreateObjectMeta(source)

	Influx(ast, source.GetInflux(), &aivenApp)
	kafkaKeyPaths := Kafka(source, ast, config, source.GetKafka(), &aivenApp)

	openSearchEnabled, err := OpenSearch(ast, source.GetOpenSearch(), &aivenApp)
	if err != nil {
		return err
	}

	redisEnabled, err := Redis(ast, source.GetRedis(), &aivenApp)
	if err != nil {
		return err
	}

	if len(kafkaKeyPaths) > 0 {
		credentialFilesVolume := pod.FromFilesSecretVolume(aivenCredentialFilesVolumeName, secretName, kafkaKeyPaths)

		ast.Volumes = append(ast.Volumes, credentialFilesVolume)
		ast.VolumeMounts = append(ast.VolumeMounts, pod.FromFilesVolumeMount(credentialFilesVolume.Name, nais_io_v1alpha1.DefaultKafkaratorMountPath, "", true))
	}

	if len(kafkaKeyPaths) > 0 || openSearchEnabled || redisEnabled {
		ast.AppendOperation(resource.OperationCreateOrUpdate, &aivenApp)
		ast.Env = append(ast.Env, makeSecretEnvVar("AIVEN_SECRET_UPDATED", aivenApp.Spec.SecretName))
	}
	return nil
}

func makeSecretEnvVar(key, secretName string) v1.EnvVar {
	return v1.EnvVar{
		Name: key,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}
