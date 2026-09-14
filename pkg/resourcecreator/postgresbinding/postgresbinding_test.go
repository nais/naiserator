package postgresbinding

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

func TestCreateCreatesOneBindingAndProjectsOnlyRequestedFiles(t *testing.T) {
	app := &nais_io_v1alpha1.Application{ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "myteam", UID: types.UID("app-uid")}, Spec: nais_io_v1alpha1.ApplicationSpec{Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{{Name: "mydb"}}}}}
	ast := resource.NewAst()
	if err := Create(app, ast); err != nil {
		t.Fatal(err)
	}
	if len(ast.Operations) != 1 {
		t.Fatalf("operations = %d, want 1", len(ast.Operations))
	}
	binding := ast.Operations[0].Resource.(*unstructured.Unstructured)
	credentials, found, err := unstructured.NestedStringSlice(binding.Object, "spec", "credentials")
	if err != nil || !found || len(credentials) != 2 || credentials[0] != "admin" || credentials[1] != "readwrite" {
		t.Errorf("binding = %#v", binding)
	}
	secretName, secretNameFound, err := unstructured.NestedString(binding.Object, "spec", "secretName")
	if err != nil || !secretNameFound || binding.GetName() != "mydb-myapp" || secretName != bindingSecretName("mydb", "myapp") {
		t.Errorf("binding = %#v", binding)
	}
	if got := ast.Volumes[0].Secret.SecretName; got != secretName {
		t.Errorf("mounted Secret = %q, want %q", got, secretName)
	}
	if got := ast.VolumeMounts; len(got) != 1 || got[0].Name != ast.Volumes[0].Name || got[0].MountPath != "/var/run/secrets/nais.io/postgres/mydb" || !got[0].ReadOnly {
		t.Errorf("volume mounts = %#v", got)
	}
	if len(ast.EnvFrom) != 0 {
		t.Errorf("EnvFrom = %#v, want none", ast.EnvFrom)
	}
	if got := ast.Volumes[0].Secret.Items; len(got) != 5 || got[0].Key != "ca.crt" || got[1].Key != "admin.tls.crt" {
		t.Errorf("projected files = %#v", got)
	}
	assertConnectionEnvironment(t, ast, "mydb", secretName, []connectionEnvironment{
		{credential: "admin"},
		{credential: "readwrite", environmentPrefix: "READWRITE_", secretKeyPrefix: "READWRITE_"},
	})
}

func TestCreateUsesPrefixedReadOnlyConnectionEnvironment(t *testing.T) {
	app := &nais_io_v1alpha1.Application{ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "myteam", UID: types.UID("app-uid")}, Spec: nais_io_v1alpha1.ApplicationSpec{Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{{Name: "reporting", Role: "read", EnvPrefix: "REPORTING_"}}}}}
	ast := resource.NewAst()
	if err := Create(app, ast); err != nil {
		t.Fatal(err)
	}

	binding := ast.Operations[0].Resource.(*unstructured.Unstructured)
	secretName, found, err := unstructured.NestedString(binding.Object, "spec", "secretName")
	if err != nil || !found {
		t.Fatalf("binding secretName = %q, found = %t, err = %v", secretName, found, err)
	}
	if got := ast.Volumes[0].Secret.Items; len(got) != 3 || got[0].Key != "ca.crt" || got[1].Key != "read.tls.crt" || got[2].Key != "read.tls.key" {
		t.Errorf("projected files = %#v", got)
	}
	if got := ast.VolumeMounts; len(got) != 1 || got[0].MountPath != "/var/run/secrets/nais.io/postgres/reporting" || !got[0].ReadOnly {
		t.Errorf("volume mounts = %#v", got)
	}
	assertConnectionEnvironment(t, ast, "reporting", secretName, []connectionEnvironment{
		{credential: "read", environmentPrefix: "REPORTING_READ_", secretKeyPrefix: "READ_"},
	})
}

type connectionEnvironment struct {
	credential        string
	environmentPrefix string
	secretKeyPrefix   string
}

func assertConnectionEnvironment(t *testing.T, ast *resource.Ast, postgres, secretName string, connections []connectionEnvironment) {
	t.Helper()
	const connectionVariableCount = 8
	if got, want := len(ast.Env), len(connections)*connectionVariableCount; got != want {
		t.Fatalf("environment variable count = %d, want %d: %#v", got, want, ast.Env)
	}

	environment := make(map[string]int, len(ast.Env))
	for i, env := range ast.Env {
		if _, exists := environment[env.Name]; exists {
			t.Errorf("duplicate environment variable %q", env.Name)
		}
		environment[env.Name] = i
	}
	get := func(name string) int {
		t.Helper()
		i, found := environment[name]
		if !found {
			t.Fatalf("environment variable %q is missing", name)
		}
		return i
	}

	for _, connection := range connections {
		for _, key := range []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGSSLMODE"} {
			env := ast.Env[get(connection.environmentPrefix+key)]
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil || env.ValueFrom.SecretKeyRef.Name != secretName || env.ValueFrom.SecretKeyRef.Key != connection.secretKeyPrefix+key {
				t.Errorf("%s = %#v, want Secret %q key %q", env.Name, env, secretName, connection.secretKeyPrefix+key)
			}
		}

		mountPath := "/var/run/secrets/nais.io/postgres/" + postgres
		wantTLSVariables := map[string]string{
			connection.environmentPrefix + "PGSSLCERT":     mountPath + "/" + connection.credential + "/tls.crt",
			connection.environmentPrefix + "PGSSLKEY":      mountPath + "/" + connection.credential + "/tls.key",
			connection.environmentPrefix + "PGSSLROOTCERT": mountPath + "/ca.crt",
		}
		for name, wantValue := range wantTLSVariables {
			env := ast.Env[get(name)]
			if env.Value != wantValue || env.ValueFrom != nil {
				t.Errorf("%s = %#v, want literal %q", name, env, wantValue)
			}
		}
	}
}

func TestBindingSecretNameDistinguishesPostgresAndWorkload(t *testing.T) {
	if bindingSecretName("a-b", "c") == bindingSecretName("a", "b-c") {
		t.Fatal("distinct Postgres uses generated the same Secret name")
	}
}

func TestCreateRejectsDuplicatePostgresUse(t *testing.T) {
	app := &nais_io_v1alpha1.Application{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}, Spec: nais_io_v1alpha1.ApplicationSpec{Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{{Name: "mydb"}, {Name: "mydb"}}}}}
	if err := Create(app, resource.NewAst()); err == nil {
		t.Fatal("expected duplicate Postgres use error")
	}
}
