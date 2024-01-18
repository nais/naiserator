package google

import (
	"fmt"
	"strconv"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/nais/naiserator/pkg/resourcecreator/pod"
)

func GcpServiceAccountName(appNamespaceHash, projectId string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", appNamespaceHash, projectId)
}

func CloudSqlProxyContainer(port int32, googleCloudSQLProxyContainerImage, projectId, instanceName string) corev1.Container {
	connectionName := fmt.Sprintf("%s:%s:%s", projectId, Region, instanceName)
	cloudSqlProxyContainerResourceSpec := nais_io_v1.ResourceRequirements{
		Limits: &nais_io_v1.ResourceSpec{
			Memory: "256Mi",
		},
		Requests: &nais_io_v1.ResourceSpec{
			Cpu:    "50m",
			Memory: "32Mi",
		},
	}

	return corev1.Container{
		Name:            "cloudsql-proxy",
		Image:           googleCloudSQLProxyContainerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{{
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		}},
		// Needs version 2.x of Cloud SQL proxy
		Command: []string{
			"/cloud-sql-proxy",
			"--max-sigterm-delay", CloudSQLProxyTermTimeout,
			"--port", strconv.Itoa(int(port)),
			connectionName,
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
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}
}
