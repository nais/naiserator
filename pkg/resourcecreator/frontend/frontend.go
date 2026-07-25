package frontend

import (
	"encoding/json"
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

// versionFromImage extracts the tag from an image reference, handling
// registry ports (registry:5000/app:tag) and digest references
// (app@sha256:... has no tag). Mirrors @nais/apm's versionFromImage so the
// generatedConfig and NAIS_APP_IMAGE resolution paths agree on the version
// for the same image (adversarial review finding, nais/naiserator#687).
func versionFromImage(image string) string {
	if at := strings.Index(image, "@"); at != -1 {
		image = image[:at]
	}
	colon := strings.LastIndex(image, ":")
	if colon == -1 || colon < strings.LastIndex(image, "/") {
		return ""
	}
	return image[colon+1:]
}

func naisConfig(source Source, telemetryURL, clusterName string) generatedConfig {
	tag := versionFromImage(source.GetEffectiveImage())

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

func naisJson(cfg generatedConfig) (string, error) {
	contents, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return "", err
	}
	return string(contents) + "\n", nil
}

// naisJs wraps the marshalled JSON as an ES module. One escaped serialization
// backs both files: values that JSON escapes (quotes, backslashes, newlines —
// reachable through the unvalidated spec.image tag) can therefore never break
// the module or make the two formats disagree (adversarial review finding,
// nais/naiserator#687).
func naisJs(jsonContents string) string {
	return "export default " + strings.TrimSuffix(jsonContents, "\n") + ";\n"
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
	cm := naisConfigMap(source, configMapName, naisJs(jsonContents), jsonContents)

	ast.AppendOperation(resource.OperationCreateOrUpdate, &cm)
	ast.PrependEnv(envVars(cfg.GetFrontendOptions().TelemetryURL)...)

	// The ES module mounts at the exact path the app chose (unchanged,
	// backward-compatible behavior).
	jsPath := frontendSpec.GeneratedConfig.MountPath
	ast.VolumeMounts = append(ast.VolumeMounts, volumeMount(jsPath, configFileName))

	// The JSON variant mounts as a sibling — but ONLY when the app follows the
	// documented convention of mounting the module as a file named nais.js.
	// Deliberately narrow (adversarial review findings, nais/naiserator#687):
	// paths are compared cleaned (so /dir//nais.js can't sneak a duplicate
	// mount past Kubernetes' exact-string uniqueness check), and directory-ish
	// or unconventional mountPaths (trailing slash, other filenames) get no
	// surprise sibling — a mount under a file path would brick the container,
	// and a sibling next to an unrelated filename lands where nothing serves
	// it. Apps that ship their own file at <dir>/nais.json will collide at
	// admission (loud, not silent); documented in the auto-configuration
	// reference.
	if cleaned := path.Clean(jsPath); path.Base(cleaned) == configFileName {
		jsonPath := path.Join(path.Dir(cleaned), jsonFileName)
		ast.VolumeMounts = append(ast.VolumeMounts, volumeMount(jsonPath, jsonFileName))
	}
	ast.Volumes = append(ast.Volumes, volume(configMapName))

	return nil
}
