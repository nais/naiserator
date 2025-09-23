package aiven

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
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

func generateAivenSecretName(appName, aivenService, generation string) (string, error) {
	prefixedName := fmt.Sprintf("aiven-%s-%s", aivenService, appName)
	year, week := time.Now().ISOWeek()
	suffix := fmt.Sprintf("%d-%d-%s", year, week, generation)
	maxLen := validation.DNS1035LabelMaxLength

	return namegen.SuffixedShortName(prefixedName, suffix, maxLen)
}

func Create(source Source, ast *resource.Ast, config Config) error {
	aivenApp := aiven_nais_io_v1.NewAivenApplicationBuilder(source.GetName(), source.GetNamespace()).
		WithSpec(aiven_nais_io_v1.AivenApplicationSpec{}).
		Build()
	aivenApp.ObjectMeta = resource.CreateObjectMeta(source)
	aivenApp.Labels["aiven.nais.io/secret-generation"] = strconv.Itoa(config.GetAivenGeneration())

	kafkaEnabled, err := Kafka(source, ast, config, source.GetKafka(), &aivenApp)
	if err != nil {
		return err
	}

	openSearchEnabled, err := OpenSearch(ast, source.GetOpenSearch(), &aivenApp)
	if err != nil {
		return err
	}

	valkeyEnabled, err := Valkey(ast, config, source, &aivenApp)
	if err != nil {
		return err
	}

	if kafkaEnabled || openSearchEnabled || valkeyEnabled {
		ast.AppendOperation(resource.OperationCreateOrUpdate, &aivenApp)
		ast.PrependEnv([]v1.EnvVar{
			// V legacy info and different for each service, depending on what got updated, when.
			makeSecretEnvVar("AIVEN_SECRET_UPDATED", aivenApp.Spec.SecretName),
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
