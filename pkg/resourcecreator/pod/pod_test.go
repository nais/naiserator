package pod

import "testing"

func TestGenerateNameFromMountPath(t *testing.T) {
	testCases := []struct {
		mountPath string
		expected  string
	}{
		{"/var/run/my-config_maps", "var-run-my-config-maps"},
		{"/var/run/secrets/nais.io/vault", "var-run-secrets-nais-io-vault"},
		{".var/run/my-config_maps_", "var-run-my-config-maps"},
	}

	for _, tc := range testCases {
		t.Run(tc.mountPath, func(t *testing.T) {
			name := generateNameFromMountPath(tc.mountPath)
			if name != tc.expected {
				t.Errorf("expected name to be '%s', was '%s'", tc.expected, name)
			}
		})
	}
}
