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
	IsInfluxCredentialsEnabled() bool
	GetAivenProject() string
}

func generateSharedAivenSecretName(name string) (string, error) {
	prefixedName := fmt.Sprintf("aiven-%s", name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d", year, week)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func Create(source Source, ast *resource.Ast, config Config) error {
	secretName, err := generateSharedAivenSecretName(source.GetName())
	if err != nil {
		return err
	}

	aivenApp := aiven_nais_io_v1.NewAivenApplicationBuilder(source.GetName(), source.GetNamespace()).
		WithSpec(aiven_nais_io_v1.AivenApplicationSpec{
			SecretName: secretName,
		}).
		Build()
	aivenApp.ObjectMeta = resource.CreateObjectMeta(source)

	kafkaKeyPaths := Kafka(source, ast, config, source.GetKafka(), &aivenApp)

	influxEnabled, err := Influx(ast, source.GetInflux(), &aivenApp, config.IsInfluxCredentialsEnabled())
	if err != nil {
		return err
	}

	openSearchEnabled, err := OpenSearch(ast, source.GetOpenSearch(), &aivenApp)
	if err != nil {
		return err
	}

	redisEnabled, err := Redis(ast, config, source, &aivenApp)
	if err != nil {
		return err
	}

	if len(kafkaKeyPaths) > 0 {
		credentialFilesVolume := pod.FromFilesSecretVolume(aivenCredentialFilesVolumeName, secretName, kafkaKeyPaths)

		ast.Volumes = append(ast.Volumes, credentialFilesVolume)
		ast.VolumeMounts = append(ast.VolumeMounts, pod.FromFilesVolumeMount(credentialFilesVolume.Name, nais_io_v1alpha1.DefaultKafkaratorMountPath, "", true))
	}

	if len(kafkaKeyPaths) > 0 || influxEnabled || openSearchEnabled || redisEnabled {
		ast.AppendOperation(resource.OperationCreateOrUpdate, &aivenApp)
		ast.EnvFrom = append(ast.EnvFrom, v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: aivenApp.Spec.SecretName,
				},
			},
		})
	}
	return nil
}
