package postgresbinding

import (
	"reflect"
	"testing"

	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCreateOmitsDeprecatedMainBranchWithoutChangingConnectionIdentity(t *testing.T) {
	for _, kind := range []string{"Application", "Naisjob"} {
		t.Run(kind, func(t *testing.T) {
			var defaultAst *resource.Ast
			for _, branch := range []string{"", "main"} {
				t.Run("branch="+branch, func(t *testing.T) {
					uses := &v1.Uses{Postgres: []v1.PostgresUse{{Name: "orders", Branch: branch, Role: "read"}}}
					meta := metav1.ObjectMeta{Name: "consumer", Namespace: "team", UID: "consumer-uid"}
					var source Source
					if kind == "Application" {
						source = &v1alpha1.Application{ObjectMeta: meta, Spec: v1alpha1.ApplicationSpec{Uses: uses}}
					} else {
						source = &v1.Naisjob{ObjectMeta: meta, Spec: v1.NaisjobSpec{Uses: uses}}
					}
					ast := resource.NewAst()
					if err := Create(source, ast); err != nil {
						t.Fatal(err)
					}
					if branch == "" {
						defaultAst = ast
					} else if !reflect.DeepEqual(defaultAst, ast) {
						t.Error("legacy main changed the binding, environment, credentials or mounted files")
					}
					binding, ok := ast.Operations[0].Resource.(*unstructured.Unstructured)
					if !ok {
						t.Fatalf("resource = %T", ast.Operations[0].Resource)
					}
					got, found, err := unstructured.NestedString(binding.Object, "spec", "branch")
					if err != nil || got != "" || found {
						t.Fatalf("binding branch = (%q, %t, %v), want omitted", got, found, err)
					}
					if binding.GetName() != bindingName("orders", "consumer") {
						t.Errorf("branch changed binding name: %q", binding.GetName())
					}
					if ast.Volumes[0].Secret.SecretName != bindingSecretName("orders", "consumer") {
						t.Error("branch changed mounted Secret")
					}
					if ast.VolumeMounts[0].MountPath != postgresMountPath("orders") {
						t.Error("branch changed mount path")
					}
				})
			}
		})
	}
}

func TestCreateRejectsNonMainBranch(t *testing.T) {
	for _, kind := range []string{"Application", "Naisjob"} {
		t.Run(kind, func(t *testing.T) {
			uses := &v1.Uses{Postgres: []v1.PostgresUse{{Name: "orders", Branch: "pr-123"}}}
			var source Source
			if kind == "Application" {
				source = &v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{Uses: uses}}
			} else {
				source = &v1.Naisjob{Spec: v1.NaisjobSpec{Uses: uses}}
			}
			ast := resource.NewAst()
			if err := Create(source, ast); err == nil {
				t.Fatal("non-main branch was silently mapped to the database")
			}
			if len(ast.Operations) != 0 {
				t.Fatal("invalid branch produced binding operations")
			}
		})
	}
}
