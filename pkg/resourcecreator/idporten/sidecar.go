package idporten

import (
	"encoding/base64"
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/keygen"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
)

func Wonderwall(port int32, targetPort int, wonderwallImage string) corev1.Container {
	var runAsUser int64 = 2
	allowPrivilegeEscalation := false

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
	return corev1.Container{
		Name:            "wonderwall",
		Image:           wonderwallImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  "WONDERWALL_UPSTREAM_HOST",
				Value: fmt.Sprintf("127.0.0.1:%d", targetPort),
			},
			{
				Name:  "WONDERWALL_REDIS",
				Value: fmt.Sprintf("nais-io-wonderwall-redis:%d", redisPort),
			},
		},
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
