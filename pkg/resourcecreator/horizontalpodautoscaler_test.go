package resourcecreator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHorizontalPodAutoscaler(t *testing.T) {
	app := getExampleApp()
	hpa := horizontalPodAutoscaler(app)

	assert.Equal(t, app.Name, hpa.Name)
	assert.Equal(t, app.Namespace, hpa.Namespace)
	assert.Equal(t, int32p(int32(app.Spec.Replicas.CpuThresholdPercentage)), hpa.Spec.TargetCPUUtilizationPercentage)
	assert.Equal(t, int32p(int32(app.Spec.Replicas.Min)), hpa.Spec.MinReplicas)
	assert.Equal(t, int32(app.Spec.Replicas.Max), hpa.Spec.MaxReplicas)
	assert.Equal(t, app.Name, hpa.Spec.ScaleTargetRef.Name)
}
