package google

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func GcpServiceAccountName(name, projectId string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", name, projectId)
}

func CloudSqlProxyContainer(port int32, googleCloudSQLProxyContainerImage, projectId, instanceName string, seccomp bool) corev1.Container {
	connectionName := fmt.Sprintf("%s:%s:%s", projectId, Region, instanceName)
	cloudSqlProxyContainerResourceSpec := nais_io_v1.ResourceRequirements{
		Limits: &nais_io_v1.ResourceSpec{
			Cpu:    "250m",
			Memory: "256Mi",
		},
		Requests: &nais_io_v1.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}

	var sc *corev1.SeccompProfile
	if seccomp {
		sc = &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		}
	}

	return corev1.Container{
		Name:            "cloudsql-proxy",
		Image:           googleCloudSQLProxyContainerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{{
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		}},
		Command: []string{
			"/cloud_sql_proxy",
			fmt.Sprintf("-term_timeout=%s", CloudSQLProxyTermTimeout),
			fmt.Sprintf("-instances=%s=tcp:%d", connectionName, port),
		},
		Resources: pod.ResourceLimits(cloudSqlProxyContainerResourceSpec),
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                pointer.Int64(2),
			RunAsGroup:               pointer.Int64(2),
			RunAsNonRoot:             pointer.Bool(true),
			Privileged:               pointer.Bool(false),
			AllowPrivilegeEscalation: pointer.Bool(false),
			ReadOnlyRootFilesystem:   pointer.Bool(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			SeccompProfile: sc,
		},
	}
}