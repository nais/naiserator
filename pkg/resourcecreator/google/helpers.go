package google

import (
	"fmt"
	"strconv"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

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

	securityContext := pod.DefaultContainerSecurityContext()
	securityContext.RunAsUser = ptr.To(int64(2))
	securityContext.RunAsGroup = ptr.To(int64(2))

	return corev1.Container{
		Name:            "cloudsql-proxy",
		Image:           googleCloudSQLProxyContainerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
		Ports: []corev1.ContainerPort{{
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		}},
		// Needs version 2.x of Cloud SQL proxy
		Command: []string{
			"/cloud-sql-proxy",
			"--max-sigterm-delay", CloudSQLProxyTermTimeout,
			"--port", strconv.Itoa(int(port)),
			"--quitquitquit",
			connectionName,
		},
		Resources:       pod.ResourceLimits(cloudSqlProxyContainerResourceSpec),
		SecurityContext: securityContext,
	}
}
