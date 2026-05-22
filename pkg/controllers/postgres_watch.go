package controllers

import (
	"context"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// postgresPartialObjectMetadata returns a PartialObjectMetadata configured to
// watch Postgres CRs. Using metadata-only watches avoids pulling full specs.
func postgresPartialObjectMetadata() *metav1.PartialObjectMetadata {
	return &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "data.nais.io/v1",
			Kind:       "Postgres",
		},
	}
}

// mapPostgresToApplications returns a map function that enqueues Applications
// in the same namespace whose spec.postgres.clusterName matches the Postgres CR name.
func mapPostgresToApplications(kube client.Client) func(ctx context.Context, obj client.Object) []ctrl.Request {
	return func(ctx context.Context, obj client.Object) []ctrl.Request {
		var apps nais_io_v1alpha1.ApplicationList
		err := kube.List(ctx, &apps, client.InNamespace(obj.GetNamespace()))
		if err != nil {
			log.Errorf("postgres watch: failed to list applications in namespace %s: %v", obj.GetNamespace(), err)
			return nil
		}

		var requests []ctrl.Request
		for _, app := range apps.Items {
			if app.Spec.Postgres != nil && app.Spec.Postgres.ClusterName == obj.GetName() {
				requests = append(requests, ctrl.Request{
					NamespacedName: client.ObjectKeyFromObject(&app),
				})
			}
		}
		return requests
	}
}

// mapPostgresToNaisjobs returns a map function that enqueues Naisjobs
// in the same namespace whose spec.postgres.clusterName matches the Postgres CR name.
func mapPostgresToNaisjobs(kube client.Client) func(ctx context.Context, obj client.Object) []ctrl.Request {
	return func(ctx context.Context, obj client.Object) []ctrl.Request {
		var jobs nais_io_v1.NaisjobList
		err := kube.List(ctx, &jobs, client.InNamespace(obj.GetNamespace()))
		if err != nil {
			log.Errorf("postgres watch: failed to list naisjobs in namespace %s: %v", obj.GetNamespace(), err)
			return nil
		}

		var requests []ctrl.Request
		for _, job := range jobs.Items {
			if job.Spec.Postgres != nil && job.Spec.Postgres.ClusterName == obj.GetName() {
				requests = append(requests, ctrl.Request{
					NamespacedName: client.ObjectKeyFromObject(&job),
				})
			}
		}
		return requests
	}
}
