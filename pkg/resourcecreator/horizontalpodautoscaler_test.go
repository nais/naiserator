package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetHorizontalPodAutoscaler(t *testing.T) {
	app := fixtures.MinimalApplication()
	err := nais.ApplyDefaults(app)
	assert.NoError(t, err)

	hpa := resourcecreator.HorizontalPodAutoscaler(app)

	assert.Equal(t, app.Name, hpa.Name)
	assert.Equal(t, app.Namespace, hpa.Namespace)
	assert.Equal(t, int32(app.Spec.Replicas.CpuThresholdPercentage), *hpa.Spec.TargetCPUUtilizationPercentage)
	assert.Equal(t, int32(app.Spec.Replicas.Min), *hpa.Spec.MinReplicas)
	assert.Equal(t, int32(app.Spec.Replicas.Max), hpa.Spec.MaxReplicas)
	assert.Equal(t, app.Name, hpa.Spec.ScaleTargetRef.Name)
}
