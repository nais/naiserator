package pod

import "testing"

func TestGenerateNameFromMountPath(t *testing.T) {
	t.Run("should generate name for normal mount path", func(t *testing.T) {
		mountPath := "/var/run/my-config_maps"
		name := generateNameFromMountPath(mountPath)
		if name != "var-run-my-config-maps" {
			t.Errorf("expected name to be 'var-run-my-config-maps', was '%s'", name)
		}
	})

	t.Run("should generate name from mount path containing dot", func(t *testing.T) {
		mountPath := "/var/run/secrets/nais.io/vault"
		name := generateNameFromMountPath(mountPath)
		if name != "var-run-secrets-nais-io-vault" {
			t.Errorf("expected name to be 'var-run-secrets-nais-io-vault', was '%s'", name)
		}
	})

	t.Run("should remove trailing and leading special chars", func(t *testing.T) {
		mountPath := ".var/run/my-config_maps_"
		name := generateNameFromMountPath(mountPath)
		if name != "var-run-my-config-maps" {
			t.Errorf("expected name to be 'var-run-my-config-maps', was '%s'", name)
		}
	})

}
