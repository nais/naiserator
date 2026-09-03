package controllers

import (
	"context"
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMapPostgresToApplications(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = nais_io_v1alpha1.AddToScheme(scheme)
	_ = nais_io_v1.AddToScheme(scheme)

	matchingApp := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "team-a",
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Postgres: &nais_io_v1.Postgres{
				ClusterName: "my-pg",
			},
		},
	}

	otherApp := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-app",
			Namespace: "team-a",
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Postgres: &nais_io_v1.Postgres{
				ClusterName: "other-pg",
			},
		},
	}

	appWithoutPostgres := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-pg-app",
			Namespace: "team-a",
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{},
	}

	appDifferentNamespace := &nais_io_v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "diff-ns-app",
			Namespace: "team-b",
		},
		Spec: nais_io_v1alpha1.ApplicationSpec{
			Postgres: &nais_io_v1.Postgres{
				ClusterName: "my-pg",
			},
		},
	}

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(matchingApp, otherApp, appWithoutPostgres, appDifferentNamespace).
		Build()

	pgObject := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pg",
			Namespace: "team-a",
		},
	}

	mapFn := mapPostgresToApplications(kube)
	requests := mapFn(context.Background(), pgObject)

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d: %v", len(requests), requests)
	}
	if requests[0].Name != "my-app" {
		t.Errorf("expected request for 'my-app', got %q", requests[0].Name)
	}
	if requests[0].Namespace != "team-a" {
		t.Errorf("expected namespace 'team-a', got %q", requests[0].Namespace)
	}
}

func TestMapPostgresToNaisjobs(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = nais_io_v1alpha1.AddToScheme(scheme)
	_ = nais_io_v1.AddToScheme(scheme)

	matchingJob := &nais_io_v1.Naisjob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "team-a",
		},
		Spec: nais_io_v1.NaisjobSpec{
			Postgres: &nais_io_v1.Postgres{
				ClusterName: "my-pg",
			},
		},
	}

	otherJob := &nais_io_v1.Naisjob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-job",
			Namespace: "team-a",
		},
		Spec: nais_io_v1.NaisjobSpec{
			Postgres: &nais_io_v1.Postgres{
				ClusterName: "other-pg",
			},
		},
	}

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(matchingJob, otherJob).
		Build()

	pgObject := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pg",
			Namespace: "team-a",
		},
	}

	mapFn := mapPostgresToNaisjobs(kube)
	requests := mapFn(context.Background(), pgObject)

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d: %v", len(requests), requests)
	}
	if requests[0].Name != "my-job" {
		t.Errorf("expected request for 'my-job', got %q", requests[0].Name)
	}
	if requests[0].Namespace != "team-a" {
		t.Errorf("expected namespace 'team-a', got %q", requests[0].Namespace)
	}
}

func TestMapPostgresToApplications_NoMatches(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	_ = nais_io_v1alpha1.AddToScheme(scheme)
	_ = nais_io_v1.AddToScheme(scheme)

	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	pgObject := &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-pg",
			Namespace: "team-a",
		},
	}

	mapFn := mapPostgresToApplications(kube)
	requests := mapFn(context.Background(), pgObject)

	if len(requests) != 0 {
		t.Fatalf("expected 0 requests, got %d: %v", len(requests), requests)
	}
}
