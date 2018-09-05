package resourcecreator

import (
	nais "github.com/nais/naiserator/api/types/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetService(t *testing.T) {
	app := &nais.Application{Spec: nais.ApplicationSpec{
		Port: nais.DefaultPort,
	}}

	svc := getService(app)

	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))
}

