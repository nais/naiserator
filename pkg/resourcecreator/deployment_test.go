package resourcecreator

import (
	nais "github.com/nais/naiserator/api/types/v1alpha1"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestGetDeployment(t *testing.T) {
	app := getExampleApp()

	deploy := getDeployment(app)

	t.Run("user settings is applied", func(t *testing.T) {
		assert.Equal(t, int32(app.Spec.Port), deploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
		assert.Equal(t, app.Name, deploy.Name)
		assert.Equal(t, app.Namespace, deploy.Namespace)
		assert.Equal(t, app.Spec.Team, deploy.Labels["team"])
		assert.Equal(t, nais.DefaultPortName, deploy.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Port.StrVal)
		assert.Equal(t, app.Spec.Healthcheck.Liveness.Path, deploy.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path)
		assert.Equal(t, int32(app.Spec.Healthcheck.Liveness.PeriodSeconds), deploy.Spec.Template.Spec.Containers[0].LivenessProbe.PeriodSeconds)
		assert.Equal(t, int32(app.Spec.Healthcheck.Liveness.Timeout), deploy.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds)
		assert.Equal(t, int32(app.Spec.Healthcheck.Liveness.FailureThreshold), deploy.Spec.Template.Spec.Containers[0].LivenessProbe.FailureThreshold)
		assert.Equal(t, int32(app.Spec.Healthcheck.Liveness.InitialDelay), deploy.Spec.Template.Spec.Containers[0].LivenessProbe.InitialDelaySeconds)
		assert.Equal(t, nais.DefaultPortName, deploy.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.StrVal)
		assert.Equal(t, app.Spec.Healthcheck.Readiness.Path, deploy.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, int32(app.Spec.Healthcheck.Readiness.PeriodSeconds), deploy.Spec.Template.Spec.Containers[0].ReadinessProbe.PeriodSeconds)
		assert.Equal(t, int32(app.Spec.Healthcheck.Readiness.Timeout), deploy.Spec.Template.Spec.Containers[0].ReadinessProbe.TimeoutSeconds)
		assert.Equal(t, int32(app.Spec.Healthcheck.Readiness.FailureThreshold), deploy.Spec.Template.Spec.Containers[0].ReadinessProbe.FailureThreshold)
		assert.Equal(t, int32(app.Spec.Healthcheck.Readiness.InitialDelay), deploy.Spec.Template.Spec.Containers[0].ReadinessProbe.InitialDelaySeconds)
		assert.Equal(t, app.Spec.Resources.Limits.Cpu, deploy.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
		assert.Equal(t, app.Spec.Resources.Limits.Memory, deploy.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		assert.Equal(t, app.Spec.Resources.Requests.Cpu, deploy.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		assert.Equal(t, app.Spec.Resources.Requests.Memory, deploy.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		assert.Equal(t, strconv.FormatBool(app.Spec.Prometheus.Enabled), deploy.Spec.Template.Annotations["prometheus.io/scrape"])
		assert.Equal(t, app.Spec.Prometheus.Path, deploy.Spec.Template.Annotations["prometheus.io/path"])
		assert.Equal(t, app.Spec.Prometheus.Port, deploy.Spec.Template.Annotations["prometheus.io/port"])
		//TODO: secrets, leaderelection
	})
}
