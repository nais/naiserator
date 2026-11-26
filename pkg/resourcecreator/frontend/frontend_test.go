package frontend

import "testing"

func Test_getImageVersion(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{
			name:  "Normal image tag",
			image: "europe-north1-docker.pkg.dev/nais-management-233d/vaktor/vaktor-frontend:2026.03.26-18.01-805aa51",
			want:  "2026.03.26-18.01-805aa51",
		},
		{
			name:  "Image tag with sha",
			image: "europe-north1-docker.pkg.dev/nais-management-233d/vaktor/vaktor-lonn:2026.11.10-12.14-6b72d1a@sha256:dc49842a559f66c99190982d50d1529bbe430469f202ecdd1a130c49fcf3a968",
			want:  "2026.11.10-12.14-6b72d1a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getImageVersion(tt.image)
			if got != tt.want {
				t.Errorf("getImageVersion(): got '%v', want '%v'", got, tt.want)
			}
		})
	}
}
