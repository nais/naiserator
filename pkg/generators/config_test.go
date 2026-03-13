package generators_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nais/naiserator/pkg/generators"
)

func TestOptions_GetIngressClasses(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   []string
	}{
		{
			name:   "Nais class",
			domain: "foo.nais.io",
			want:   []string{"nais"},
		},
		{
			name:   "External class",
			domain: "foo.external.nais.io",
			want:   []string{"nais-external"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var o generators.Options
			o.Config.DomainIngressClassMap = map[string][]string{
				"nais.io":          {"nais"},
				"external.nais.io": {"nais-external"},
			}
			got, gotErr := o.GetIngressClasses(tt.domain)
			if gotErr != nil {
				t.Errorf("GetIngressClasses() failed: %v", gotErr)
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("GetIngressClasses() (-want +got):\n%s", diff)
			}
		})
	}
}
