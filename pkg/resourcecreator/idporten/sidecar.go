package idporten

import (
	"encoding/base64"
	"fmt"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/keygen"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
)

func Wonderwall(port int32, targetPort int, wonderwallImage string, naisIdporten *nais_io_v1.IDPorten, naisIngresses []nais_io_v1.Ingress) corev1.Container {
	var runAsUser int64 = 2
	allowPrivilegeEscalation := false

	ingresses := make([]string, 0)
	for _, ingress := range naisIngresses {
		ingresses = append(ingresses, string(ingress))
	}

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
			Name:  "WONDERWALL_REDIS",
			Value: fmt.Sprintf("%s:%d", RedisName, redisPort),
		},
		{
			Name:  "WONDERWALL_INGRESSES",
			Value: strings.Join(ingresses, ","),
		},
		{
			Name:  "WONDERWALL_IDPORTEN_SECURITY_LEVEL_ENABLED",
			Value: "true",
		},
		{
			Name:  "WONDERWALL_IDPORTEN_LOCALE_ENABLED",
			Value: "true",
		},
	}

	if len(naisIdporten.Sidecar.Level) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_IDPORTEN_SECURITY_LEVEL_VALUE",
			Value: naisIdporten.Sidecar.Level,
		})
	}

	if len(naisIdporten.Sidecar.Locale) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_IDPORTEN_LOCALE_VALUE",
			Value: naisIdporten.Sidecar.Locale,
		})
	}

	if len(naisIdporten.PostLogoutRedirectURIs) > 0 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "WONDERWALL_IDPORTEN_POST_LOGOUT_REDIRECT_URI",
			Value: string(naisIdporten.PostLogoutRedirectURIs[0]),
		})
	}

	return corev1.Container{
		Name:            "wonderwall",
		Image:           wonderwallImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVars,
		Ports: []corev1.ContainerPort{{
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
			Name:          "redis",
		}},
		Resources: pod.ResourceLimits(resourcesSpec),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &runAsUser,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
	}
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
