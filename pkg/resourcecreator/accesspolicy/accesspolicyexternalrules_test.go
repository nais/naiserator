package accesspolicy_test

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/accesspolicy"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestToAccessPolicyExternalRule(t *testing.T) {
	t.Run("list of hosts should return list of access policy external rule", func(t *testing.T) {
		hosts := []string{
			"https://some-host",
			"https://some-other-host",
		}
		rules := accesspolicy.ToAccessPolicyExternalRules(hosts)
		assert.Len(t, rules, 2)
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "https://some-host"})
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "https://some-other-host"})
	})
}

func TestMergeExternalRules(t *testing.T) {
	t.Run("app external outbound rules correctly merged with additional hosts", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		additionalRules := accesspolicy.ToAccessPolicyExternalRules([]string{
			"some-host.test",
			"some-other-host.test",
			"some-other-host-2.test",
		})

		app.Spec.AccessPolicy.Outbound.External = append(app.Spec.AccessPolicy.Outbound.External,
			nais_io_v1.AccessPolicyExternalRule{Host: "some-host.test"},
			nais_io_v1.AccessPolicyExternalRule{Host: "some-other-host-3.test"},
			nais_io_v1.AccessPolicyExternalRule{Host: "some-other-host-4.test", Ports: []nais_io_v1.AccessPolicyPortRule{{Port: 1337}}},
		)

		rules := accesspolicy.MergeExternalRules(app, additionalRules...)

		assert.Len(t, rules, 5)
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "some-host.test"})
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "some-other-host.test"})
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "some-other-host-2.test"})
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "some-other-host-3.test"})
		assert.Contains(t, rules, nais_io_v1.AccessPolicyExternalRule{Host: "some-other-host-4.test", Ports: []nais_io_v1.AccessPolicyPortRule{{Port: 1337}}})
	})
}
