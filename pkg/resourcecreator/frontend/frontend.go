package frontend

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/event/generator"
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
}

const volumeName = "frontend-config"
const configFileName = "nais.js"
const configMapSuffix = "-frontend-config-js"

var naisJsTemplate = `
export default {
	telemetryCollectorURL: '%s',
	app: {
		name: '%s',
		version: '%s'
	}
};
`

func naisJs(source Source, telemetryURL string) string {
	img := generator.ContainerImage(source.GetEffectiveImage())
	return fmt.Sprintf(naisJsTemplate, telemetryURL, source.GetName(), img.GetTag())
}

func naisJsConfigMap(source Source, name, contents string) corev1.ConfigMap {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
		Data: map[string]string{
			configFileName: contents,
		},
	}
}

func volumeMount(mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   configFileName,
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

	naisJsContents := naisJs(source, cfg.GetFrontendOptions().TelemetryURL)
	cm := naisJsConfigMap(source, configMapName, naisJsContents)

	ast.AppendOperation(resource.OperationCreateOrUpdate, &cm)
	ast.PrependEnv(envVars(cfg.GetFrontendOptions().TelemetryURL)...)
	ast.VolumeMounts = append(ast.VolumeMounts, volumeMount(frontendSpec.GeneratedConfig.MountPath))
	ast.Volumes = append(ast.Volumes, volume(configMapName))

	return nil
}
