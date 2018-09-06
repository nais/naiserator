package resourcecreator

import (
	nais "github.com/nais/naiserator/api/types/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetService(t *testing.T) {
	svc := getService(getExampleApp())

	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))
}
