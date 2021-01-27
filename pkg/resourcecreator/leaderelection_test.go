package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestLeaderElection(t *testing.T) {
	t.Run("check that role and rolebinding is correct", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		role := resourcecreator.LeaderElectionRole(app)
		rolebinding := resourcecreator.LeaderElectionRoleBinding(app)

		assert.Equal(t, app.Name, role.Name)
		assert.Len(t, role.Rules, 1)
		assert.Equal(t, []string{app.Name}, role.Rules[0].ResourceNames)

		assert.Equal(t, app.Name, rolebinding.Name)
		assert.Equal(t, role.Name, rolebinding.RoleRef.Name)
		assert.Equal(t, app.Name, rolebinding.Subjects[0].Name)
		assert.Equal(t, app.Namespace, rolebinding.Subjects[0].Namespace)
		assert.Equal(t, "ServiceAccount", rolebinding.Subjects[0].Kind)
	})
}
