package generators

import (
	"context"
	"strings"
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/postgres"
	"github.com/nais/pgrator/pkg/api/datav1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPreparePostgresEngineResolution(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		status     *datav1.PostgresStatus
		wantEngine string
		wantErr    string
	}{
		{
			name:       "uses engine when present",
			status:     &datav1.PostgresStatus{Engine: postgres.EngineCNPG},
			wantEngine: postgres.EngineCNPG,
		},
		{
			name:    "returns retry error when status is missing",
			status:  nil,
			wantErr: "waiting for pgrator to set engine in status on team-a/my-pg-cluster; will retry",
		},
		{
			name:    "returns retry error when engine is missing",
			status:  &datav1.PostgresStatus{},
			wantErr: "waiting for pgrator to set engine in status on team-a/my-pg-cluster; will retry",
		},
		{
			name:    "returns error for unknown engine",
			status:  &datav1.PostgresStatus{Engine: "unknown"},
			wantErr: "unknown postgres engine: unknown",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			if err := datav1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add datav1 to scheme: %v", err)
			}

			pg := &datav1.Postgres{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-pg-cluster",
					Namespace: "team-a",
				},
				Status: tt.status,
			}

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(pg).
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
