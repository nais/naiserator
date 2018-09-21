package resourcecreator

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGetService(t *testing.T) {
	svc := getService(getExampleApp())

	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))
}
