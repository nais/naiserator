package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGetService(t *testing.T) {
	svc := resourcecreator.Service(fixtures.Application())

	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))
}
