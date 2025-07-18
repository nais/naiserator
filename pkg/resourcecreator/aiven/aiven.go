package aiven

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

var namePattern = regexp.MustCompile("[^a-z0-9]")

type Source interface {
	resource.Source
	GetKafka() *nais_io_v1.Kafka
	GetOpenSearch() *nais_io_v1.OpenSearch
	GetValkey() []nais_io_v1.Valkey
}

type Config interface {
	IsKafkaratorEnabled() bool
	GetAivenProject() string
	GetAivenGeneration() int
}

// TODO: Remove once all aiven secrets are per service
func generateSharedAivenSecretName(name string, generation int) (string, error) {
	prefixedName := fmt.Sprintf("aiven-%s", name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d-%d", year, week, generation%10)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func generateAivenSecretName(name, service, generation string) (string, error) {
	prefixedName := fmt.Sprintf("aiven-%s-%s", service, name)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d-%s", year, week, generation)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func Create(source Source, ast *resource.Ast, config Config) error {
	secretName, err := generateSharedAivenSecretName(source.GetName(), config.GetAivenGeneration())
	if err != nil {
		return err
	}

	aivenApp := aiven_nais_io_v1.NewAivenApplicationBuilder(source.GetName(), source.GetNamespace()).
		WithSpec(aiven_nais_io_v1.AivenApplicationSpec{
			SecretName: secretName,
		}).
		Build()
	aivenApp.ObjectMeta = resource.CreateObjectMeta(source)
	aivenApp.ObjectMeta.Labels["aiven.nais.io/secret-generation"] = strconv.Itoa(config.GetAivenGeneration())

	kafkaKeyPaths := Kafka(source, ast, config, source.GetKafka(), &aivenApp)

	openSearchEnabled, err := OpenSearch(ast, source.GetOpenSearch(), &aivenApp)
	if err != nil {
		return err
	}

	valkeyEnabled, err := Valkey(ast, config, source, &aivenApp)
	if err != nil {
		return err
	}

	if len(kafkaKeyPaths) > 0 {
		credentialFilesVolume := pod.FromFilesSecretVolume(aivenCredentialFilesVolumeName, secretName, kafkaKeyPaths)

		ast.Volumes = append(ast.Volumes, credentialFilesVolume)
		ast.VolumeMounts = append(ast.VolumeMounts, pod.FromFilesVolumeMount(credentialFilesVolume.Name, nais_io_v1alpha1.DefaultKafkaratorMountPath, "", true))
	}

	if len(kafkaKeyPaths) > 0 || openSearchEnabled || valkeyEnabled {
		ast.AppendOperation(resource.OperationCreateOrUpdate, &aivenApp)
		ast.PrependEnv([]v1.EnvVar{
			makeSecretEnvVar("AIVEN_SECRET_UPDATED", aivenApp.Spec.SecretName),
			makeOptionalSecretEnvVar("AIVEN_CA", aivenApp.Spec.SecretName),
		}...)
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

func makeOptionalSecretEnvVar(key, secretName string) v1.EnvVar {
	optional := true
	return v1.EnvVar{
		Name: key,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key:      key,
				Optional: &optional,
			},
		},
	}
}

func envVarSuffix(instanceName string) string {
	return strings.ToUpper(namePattern.ReplaceAllString(instanceName, "_"))
}
