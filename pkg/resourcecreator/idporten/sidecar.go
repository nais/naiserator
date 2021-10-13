package idporten

import (
	"encoding/base64"
	"fmt"
	"strconv"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/keygen"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
)

const (
	WonderwallPortName        = "wonderwall"
	WonderwallMetricsPortName = "ww-metrics"
	WonderwallPort            = 7564
	WonderwallMetricsPort     = 7565
)

func Wonderwall(app *nais_io_v1alpha1.Application, wonderwallImage string) (*corev1.Container, error) {
	var runAsUser int64 = 2
	allowPrivilegeEscalation := false

	app.Labels["aiven"] = "enabled"

	targetPort := app.Spec.Port

	naisIdPorten := app.Spec.IDPorten
	naisIngresses := app.Spec.Ingresses

	resourcesSpec := nais_io_v1.ResourceRequirements{
		Limits: &nais_io_v1.ResourceSpec{
			Cpu:    "250m",
			Memory: "256Mi",
		},
		Requests: &nais_io_v1.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "WONDERWALL_UPSTREAM_HOST",
			Value: fmt.Sprintf("127.0.0.1:%d", targetPort),
		},
		{
			Name:  "WONDERWALL_INGRESS",
			Value: string(naisIngresses[0]),
		},
		{
			Name:  "WONDERWALL_IDPORTEN_SECURITY_LEVEL_ENABLED",
			Value: "true",
		},
		{
			Name:  "WONDERWALL_IDPORTEN_LOCALE_ENABLED",
			Value: "true",
		},
		{
			Name:  "WONDERWALL_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", WonderwallPort),
		},
		{
			Name:  "WONDERWALL_METRICS_BIND_ADDRESS",
			Value: fmt.Sprintf("0.0.0.0:%d", WonderwallMetricsPort),
		},
	}

	if len(naisIdPorten.Sidecar.Level) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_IDPORTEN_SECURITY_LEVEL_VALUE",
			Value: naisIdPorten.Sidecar.Level,
		})
	}

	if len(naisIdPorten.Sidecar.Locale) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_IDPORTEN_LOCALE_VALUE",
			Value: naisIdPorten.Sidecar.Locale,
		})
	}

	if len(naisIdPorten.PostLogoutRedirectURIs) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_IDPORTEN_POST_LOGOUT_REDIRECT_URI",
			Value: string(naisIdPorten.PostLogoutRedirectURIs[0]),
		})
	}

	if len(naisIdPorten.Sidecar.ErrorPath) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_ERROR_PATH",
			Value: app.Spec.IDPorten.Sidecar.ErrorPath,
		})
	}

	if naisIdPorten.Sidecar.AutoLogin {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_AUTO_LOGIN",
			Value: strconv.FormatBool(app.Spec.IDPorten.Sidecar.AutoLogin),
		})
	}

	return &corev1.Container{
		Name:            "wonderwall",
		Image:           wonderwallImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars,
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(WonderwallPort),
			Protocol:      corev1.ProtocolTCP,
			Name:          WonderwallPortName,
		}, {
			ContainerPort: int32(WonderwallMetricsPort),
			Protocol:      corev1.ProtocolTCP,
			Name:          WonderwallMetricsPortName,
		}},
		Resources: pod.ResourceLimits(resourcesSpec),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsUser,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
	}, nil
}

func WonderwallSecret(source resource.Source, secretName string) (*corev1.Secret, error) {
	key, err := keygen.Keygen(32)
	if err != nil {
		return nil, fmt.Errorf("generating secret key: %w", err)
	}

	secrets := map[string]string{
		"WONDERWALL_ENCRYPTION_KEY": base64.StdEncoding.EncodeToString(key),
	}

	objectMeta := resource.CreateObjectMeta(source)
	sec := secret.OpaqueSecret(objectMeta, secretName, secrets)

	return sec, nil
}
