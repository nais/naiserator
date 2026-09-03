package postgresbinding

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	pgrator_v1 "github.com/nais/pgrator/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name         string
		postgres     nais_io_v1.PostgresUse
		wantRoles    []pgrator_v1.PostgresBindingRole
		wantEnvNames []string
		wantCASecret string
	}{
		{
			name:     "default creates admin and runtime credentials",
			postgres: nais_io_v1.PostgresUse{Name: "mydb"},
			wantRoles: []pgrator_v1.PostgresBindingRole{
				pgrator_v1.PostgresBindingRoleAdmin,
				pgrator_v1.PostgresBindingRoleReadWrite,
			},
			wantEnvNames: []string{
				"PGSSLCERT", "PGSSLKEY", "PGSSLROOTCERT",
				"READWRITE_PGSSLCERT", "READWRITE_PGSSLKEY", "READWRITE_PGSSLROOTCERT",
			},
			wantCASecret: "pg-mydb-ca",
		},
		{
			name:      "literal prefix is prepended",
			postgres:  nais_io_v1.PostgresUse{Name: "reporting", Role: "read", EnvPrefix: "REPORTING_"},
			wantRoles: []pgrator_v1.PostgresBindingRole{pgrator_v1.PostgresBindingRoleRead},
			wantEnvNames: []string{
				"REPORTING_READ_PGSSLCERT", "REPORTING_READ_PGSSLKEY", "REPORTING_READ_PGSSLROOTCERT",
			},
			wantCASecret: "pg-reporting-ca",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &nais_io_v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "myteam", UID: types.UID("app-uid")},
				Spec: nais_io_v1alpha1.ApplicationSpec{
					Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{tt.postgres}},
				},
			}
			ast := resource.NewAst()

			err := Create(app, ast)
			require.NoError(t, err)
			require.Len(t, ast.Operations, len(tt.wantRoles))
			require.Len(t, ast.EnvFrom, len(tt.wantRoles))
			require.Len(t, ast.Volumes, len(tt.wantRoles)+1)
			require.Len(t, ast.VolumeMounts, len(tt.wantRoles)+1)

			for i, wantRole := range tt.wantRoles {
				binding, ok := ast.Operations[i].Resource.(*pgrator_v1.PostgresBinding)
				require.True(t, ok)
				assert.Equal(t, resource.OperationCreateOrUpdate, ast.Operations[i].Operation)
				assert.Equal(t, tt.postgres.Name+"-myapp-"+string(wantRole), binding.Name)
				assert.Equal(t, binding.Name+"-client-cert", binding.Spec.SecretName)
				assert.Equal(t, tt.postgres.Name, binding.Spec.Postgres)
				require.NotNil(t, binding.Spec.Consumer.Workload)
				assert.Equal(t, "myapp", binding.Spec.Consumer.Workload.Name)
				assert.Equal(t, pgrator_v1.PostgresBindingWorkloadTypeApplication, binding.Spec.Consumer.Workload.Type)
				assert.Equal(t, wantRole, binding.Spec.Role)
				assert.Equal(t, tt.postgres.EnvPrefix, ast.EnvFrom[i].Prefix)
				assert.Equal(t, binding.Name, ast.EnvFrom[i].SecretRef.Name)
			}

			gotEnvNames := make([]string, 0, len(ast.Env))
			for _, env := range ast.Env {
				gotEnvNames = append(gotEnvNames, env.Name)
			}
			assert.Equal(t, tt.wantEnvNames, gotEnvNames)

			caVolume := ast.Volumes[0]
			assert.Equal(t, tt.wantCASecret, caVolume.Secret.SecretName)
			assert.Equal(t, int32(0o440), *caVolume.Secret.DefaultMode)
			require.Len(t, caVolume.Secret.Items, 1)
			assert.Equal(t, "ca.crt", caVolume.Secret.Items[0].Key)
			assert.Equal(t, "ca.crt", caVolume.Secret.Items[0].Path)

			for _, volume := range ast.Volumes[1:] {
				assert.Equal(t, int32(0o440), *volume.Secret.DefaultMode)
				require.Len(t, volume.Secret.Items, 2)
			}
		})
	}
}

func TestCreateNaisjobBinding(t *testing.T) {
	job := &nais_io_v1.Naisjob{
		ObjectMeta: metav1.ObjectMeta{Name: "myjob", Namespace: "myteam", UID: types.UID("job-uid")},
		Spec: nais_io_v1.NaisjobSpec{
			Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{{Name: "mydb", Role: "readwrite"}}},
		},
	}
	ast := resource.NewAst()

	err := Create(job, ast)
	require.NoError(t, err)
	require.Len(t, ast.Operations, 1)
	binding := ast.Operations[0].Resource.(*pgrator_v1.PostgresBinding)
	require.NotNil(t, binding.Spec.Consumer.Workload)
	assert.Equal(t, pgrator_v1.PostgresBindingWorkloadTypeJob, binding.Spec.Consumer.Workload.Type)
}

func TestCreateRejectsUnsupportedRole(t *testing.T) {
	app := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "myteam"},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{{Name: "mydb", Role: "owner"}}},
		},
	}

	err := Create(app, resource.NewAst())
	require.EqualError(t, err, `unsupported PostgresBinding role "owner"`)
}
