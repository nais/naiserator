package postgresbinding

import (
	"net/url"
	"strings"
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestJDBCConnectionUsesCredentialRoleAndPrefix(t *testing.T) {
	for _, tt := range []struct {
		name        string
		role        string
		envPrefix   string
		credentials []string
	}{
		{"default", "", "", []string{"admin", "readwrite"}},
		{"prefixed read", "read", "ORDERS_", []string{"read"}},
		{"readwrite", "readwrite", "", []string{"readwrite"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := &v1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "team", UID: "consumer-uid"},
				Spec: v1alpha1.ApplicationSpec{Uses: &v1.Uses{Postgres: []v1.PostgresUse{
					{Name: "orders", Role: tt.role, EnvPrefix: tt.envPrefix},
				}}},
			}
			ast := resource.NewAst()
			if err := Create(app, ast); err != nil {
				t.Fatal(err)
			}
			assertJDBCConnections(t, ast, "orders", tt.envPrefix, tt.credentials)
		})
	}
}

func TestJDBCConnectionsForMultipleDatabasesStaySeparate(t *testing.T) {
	app := &v1alpha1.Application{ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "team", UID: "consumer-uid"}, Spec: v1alpha1.ApplicationSpec{Uses: &v1.Uses{Postgres: []v1.PostgresUse{
		{Name: "orders", Role: "readwrite", EnvPrefix: "ORDERS_"},
		{Name: "reporting", Role: "read", EnvPrefix: "REPORTING_"},
	}}}}
	ast := resource.NewAst()
	if err := Create(app, ast); err != nil {
		t.Fatal(err)
	}
	assertJDBCConnections(t, ast, "orders", "ORDERS_", []string{"readwrite"})
	assertJDBCConnections(t, ast, "reporting", "REPORTING_", []string{"read"})
}

func assertJDBCConnections(t *testing.T, ast *resource.Ast, database, envPrefix string, credentials []string) {
	t.Helper()
	secretName := bindingSecretName(database, "consumer")
	files := make(map[string]string)
	for _, volume := range ast.Volumes {
		if volume.Secret == nil || volume.Secret.SecretName != secretName {
			continue
		}
		if volume.Secret.DefaultMode == nil || *volume.Secret.DefaultMode != 0o440 {
			t.Fatal("private keys must retain the existing restricted file mode")
		}
		if volume.Secret.Optional != nil && *volume.Secret.Optional {
			t.Fatal("credential files must not be optional")
		}
		for _, item := range volume.Secret.Items {
			files[item.Path] = item.Key
		}
	}
	if len(files) != 1+3*len(credentials) || files["ca.crt"] != "ca.crt" {
		t.Fatalf("projected files = %v", files)
	}
	for _, credential := range credentials {
		for _, name := range []string{"tls.crt", "tls.key", "tls.pk8"} {
			if files[credential+"/"+name] != credential+"."+name {
				t.Errorf("missing or incorrect %s/%s projection", credential, name)
			}
		}
		prefix := envPrefix
		secretPrefix := ""
		user := "app"
		if credential != "admin" {
			secretPrefix = strings.ToUpper(credential) + "_"
			prefix += secretPrefix
			user = "consumer-" + credential
		}
		// Use a host that can differ from the logical database name: branch
		// selection must come from the binding Secret, not a hardcoded URL.
		values := map[string]string{"PGHOST": "selected-branch-pooler.team", "PGPORT": "5432", "PGDATABASE": "app", "PGUSER": user, "PGSSLMODE": "verify-full"}
		replacements := make([]string, 0, len(values)*2)
		seen := make(map[string]bool)
		foundURL := false
		for _, env := range ast.Env {
			key := strings.TrimPrefix(env.Name, prefix)
			if value, ok := values[key]; ok && env.Name == prefix+key {
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil || env.ValueFrom.SecretKeyRef.Name != secretName || env.ValueFrom.SecretKeyRef.Key != secretPrefix+key {
					t.Fatalf("%s does not reference its own binding credential", env.Name)
				}
				seen[key] = true
				replacements = append(replacements, "$("+env.Name+")", value)
			}
			if env.Name != prefix+"PGJDBCURL" {
				continue
			}
			if foundURL || env.ValueFrom != nil || len(seen) != len(values) {
				t.Fatal("JDBC URL must appear once, after its Secret-backed dependencies")
			}
			foundURL = true
			for _, dependency := range []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER"} {
				if !strings.Contains(env.Value, "$("+prefix+dependency+")") {
					t.Errorf("URL does not reference %s%s", prefix, dependency)
				}
			}
			expanded := strings.NewReplacer(replacements...).Replace(env.Value)
			if !strings.HasPrefix(expanded, "jdbc:postgresql://") || strings.Contains(expanded, "$(") {
				t.Fatalf("invalid or unresolved JDBC URL %q", expanded)
			}
			parsed, err := url.Parse(strings.TrimPrefix(expanded, "jdbc:"))
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Host != "selected-branch-pooler.team:5432" || parsed.Path != "/app" || parsed.User != nil {
				t.Errorf("unexpected connection target %s", expanded)
			}
			path := "/var/run/secrets/nais.io/postgres/" + database
			query := parsed.Query()
			want := map[string]string{"user": user, "sslmode": "verify-full", "sslrootcert": path + "/ca.crt", "sslcert": path + "/" + credential + "/tls.crt", "sslkey": path + "/" + credential + "/tls.pk8"}
			if len(query) != len(want) {
				t.Errorf("unexpected connection parameters: %v", query)
			}
			for name, value := range want {
				if got := query[name]; len(got) != 1 || got[0] != value {
					t.Errorf("JDBC parameter %s = %v, want %q", name, got, value)
				}
			}
		}
		if !foundURL {
			t.Errorf("missing %sPGJDBCURL", prefix)
		}
	}
}
