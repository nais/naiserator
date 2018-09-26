package resourcecreator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceAccount(t *testing.T) {
	app := getExampleApp()
	svcAcc := serviceAccount(app)

	assert.Equal(t, app.Name, svcAcc.Name)
	assert.Equal(t, app.Namespace, svcAcc.Namespace)
}
