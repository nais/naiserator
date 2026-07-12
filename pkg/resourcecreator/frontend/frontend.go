package frontend

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type Source interface {
	resource.Source
	GetFrontend() *nais_io_v1.Frontend
	GetEffectiveImage() string
}

type Config interface {
	GetFrontendOptions() config.Frontend
	GetClusterName() string
}

const (
	volumeName      = "frontend-config"
	configFileName  = "nais.js"
	jsonFileName    = "nais.json"
	configMapSuffix = "-frontend-config-js"
)

// generatedConfig is the frontend config contract consumed by the @nais/apm
// SDK (nais/grafana-apm-app#134): bump SchemaVersion when the shape changes.
// It is emitted twice with identical content: as an ES module (nais.js, for
// import) and as JSON (nais.json, for fetch from a served web root).
type generatedConfig struct {
	SchemaVersion         int                `json:"schemaVersion"`
	TelemetryCollectorURL string             `json:"telemetryCollectorURL"`
	App                   generatedConfigApp `json:"app"`
	Environment           string             `json:"environment"`
}

type generatedConfigApp struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

var naisJsTemplate = `
export default {
	schemaVersion: %d,
	telemetryCollectorURL: '%s',
	app: {
		name: '%s',
		namespace: '%s',
		version: '%s'
	},
	environment: '%s'
};
`

func naisConfig(source Source, telemetryURL, clusterName string) generatedConfig {
	imageName := strings.Split(source.GetEffectiveImage(), ":")
	tag := ""
	if len(imageName) == 2 {
		tag = imageName[1]
	}

	return generatedConfig{
		SchemaVersion:         1,
		TelemetryCollectorURL: telemetryURL,
		App: generatedConfigApp{
			Name:      source.GetName(),
			Namespace: source.GetNamespace(),
			Version:   tag,
		},
		Environment: clusterName,
	}
}

func naisJs(cfg generatedConfig) string {
	return fmt.Sprintf(naisJsTemplate,
		cfg.SchemaVersion, cfg.TelemetryCollectorURL, cfg.App.Name, cfg.App.Namespace, cfg.App.Version, cfg.Environment)
}

func naisJson(cfg generatedConfig) (string, error) {
	contents, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return "", err
	}
	return string(contents) + "\n", nil
}

func naisConfigMap(source Source, name, jsContents, jsonContents string) corev1.ConfigMap {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
		Data: map[string]string{
			configFileName: jsContents,
			jsonFileName:   jsonContents,
		},
	}
}

func volumeMount(mountPath, subPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   subPath,
		ReadOnly:  true,
	}
}

// Configures a Volume to mount files from the CA bundle ConfigMap.
func volume(configMapName string) corev1.Volume {
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

const envVarTelemetryURL = "NAIS_FRONTEND_TELEMETRY_COLLECTOR_URL"

func envVars(telemetryURL string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  envVarTelemetryURL,
			Value: telemetryURL,
		},
	}
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	frontendSpec := source.GetFrontend()
	if frontendSpec == nil || frontendSpec.GeneratedConfig == nil {
		return nil
	}

	baseName := source.GetName() + configMapSuffix
	configMapName, err := namegen.ShortName(baseName, validation.DNS1035LabelMaxLength)
	if err != nil {
		return err
	}

	generated := naisConfig(source, cfg.GetFrontendOptions().TelemetryURL, cfg.GetClusterName())
	jsonContents, err := naisJson(generated)
	if err != nil {
		return err
	}
	cm := naisConfigMap(source, configMapName, naisJs(generated), jsonContents)

	ast.AppendOperation(resource.OperationCreateOrUpdate, &cm)
	ast.PrependEnv(envVars(cfg.GetFrontendOptions().TelemetryURL)...)

	// The ES module mounts at the exact path the app chose; the JSON variant
	// mounts as its sibling (same directory), so pointing mountPath into a
	// served web root exposes both files.
	jsPath := frontendSpec.GeneratedConfig.MountPath
	ast.VolumeMounts = append(ast.VolumeMounts, volumeMount(jsPath, configFileName))
	if jsonPath := path.Join(path.Dir(jsPath), jsonFileName); jsonPath != jsPath {
		ast.VolumeMounts = append(ast.VolumeMounts, volumeMount(jsonPath, jsonFileName))
	}
	ast.Volumes = append(ast.Volumes, volume(configMapName))

	return nil
}
