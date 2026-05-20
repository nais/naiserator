package generators

import (
	"context"
	"strings"
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/postgres"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPreparePostgresEngineResolution(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		annotations map[string]string
		wantEngine  string
		wantErr     string
	}{
		{
			name: "uses active-engine when present",
			annotations: map[string]string{
				postgres.ActiveEngineAnnotation: postgres.EngineCNPG,
			},
			wantEngine: postgres.EngineCNPG,
		},
		{
			name:        "returns retry error when active-engine is missing",
			annotations: nil,
			wantErr:     "waiting for pgrator to set active-engine annotation",
		},
		{
			name: "returns error for unknown active-engine",
			annotations: map[string]string{
				postgres.ActiveEngineAnnotation: "unknown",
			},
			wantErr: "unknown postgres engine: unknown",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			scheme.AddKnownTypeWithName(schema.GroupVersionKind{
				Group:   "data.nais.io",
				Version: "v1",
				Kind:    "Postgres",
			}, &metav1.PartialObjectMetadata{})

			pg := &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Postgres",
					APIVersion: "data.nais.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-pg-cluster",
					Namespace:   "team-a",
					Annotations: tt.annotations,
				},
			}

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(pg).
				Build()

			source := &nais_io_v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "team-a",
				},
				Spec: nais_io_v1alpha1.ApplicationSpec{
					Postgres: &nais_io_v1.Postgres{
						ClusterName: "my-pg-cluster",
					},
				},
			}
			opts := &Options{}

			err := preparePostgres(context.Background(), source, cl, opts)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("preparePostgres() error = %v", err)
			}
			if opts.PostgresClusterEngine != tt.wantEngine {
				t.Fatalf("engine = %q, want %q", opts.PostgresClusterEngine, tt.wantEngine)
			}
		})
	}
}
