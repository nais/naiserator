package google

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
)

func GcpServiceAccountName(app *nais_io_v1alpha1.Application, projectId string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", app.CreateAppNamespaceHash(), projectId)
}

func CloudSqlProxyContainer(sqlInstance nais_io_v1alpha1.CloudSqlInstance, port int32, projectId string) corev1.Container {
	connectionName := fmt.Sprintf("%s:%s:%s", projectId, Region, sqlInstance.Name)
	var runAsUser int64 = 2
	allowPrivilegeEscalation := false
	cloudSqlProxyContainerResourceSpec := nais_io_v1alpha1.ResourceRequirements{
		Limits: &nais_io_v1alpha1.ResourceSpec{
			Cpu:    "250m",
			Memory: "256Mi",
		},
		Requests: &nais_io_v1alpha1.ResourceSpec{
			Cpu:    "20m",
			Memory: "32Mi",
		},
	}
	return corev1.Container{
		Name:            "cloudsql-proxy",
		Image:           viper.GetString(config.GoogleCloudSQLProxyContainerImage),
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
			RunAsUser:                &runAsUser,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
	}
}
