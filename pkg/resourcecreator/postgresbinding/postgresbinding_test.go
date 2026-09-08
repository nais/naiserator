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
	if err != nil || !found || binding.GetName() != "mydb-myapp" || len(credentials) != 2 || credentials[0] != "admin" || credentials[1] != "readwrite" {
		t.Errorf("binding = %#v", binding)
	}
	if len(ast.EnvFrom) != 0 {
		t.Errorf("EnvFrom = %#v, want none", ast.EnvFrom)
	}
	if got := ast.Volumes[0].Secret.Items; len(got) != 5 || got[0].Key != "ca.crt" || got[1].Key != "admin.tls.crt" {
		t.Errorf("projected files = %#v", got)
	}
}

func TestCreateRejectsDuplicatePostgresUse(t *testing.T) {
	app := &nais_io_v1alpha1.Application{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}, Spec: nais_io_v1alpha1.ApplicationSpec{Uses: &nais_io_v1.Uses{Postgres: []nais_io_v1.PostgresUse{{Name: "mydb"}, {Name: "mydb"}}}}}
	if err := Create(app, resource.NewAst()); err == nil {
		t.Fatal("expected duplicate Postgres use error")
	}
}
