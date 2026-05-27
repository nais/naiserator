package controllers

import (
	"context"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ApplicationReconciler struct {
	synchronizer Interface
}

func NewAppReconciler(synchronizer Interface) *ApplicationReconciler {
	return &ApplicationReconciler{
		synchronizer: synchronizer,
	}
}

// +kubebuilder:rbac:groups=nais.io,resources=Applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=Applications/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.synchronizer.Reconcile(ctx, req, &nais_io_v1alpha1.Application{})
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager, cfg *config.Config, opts ...Option) error {
	controllerBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1alpha1.Application{}).
		Watches(&nais_io_v1.Image{}, handler.EnqueueRequestsFromMapFunc(mapImageToApplicationOrNaisjob))

	if cfg.Features.PostgresOperator {
		controllerBuilder = controllerBuilder.
			WatchesMetadata(
				postgresMetadata,
				handler.EnqueueRequestsFromMapFunc(mapPostgresToApplications(mgr.GetClient())),
				builder.WithPredicates(predicate.AnnotationChangedPredicate{}),
			)
	}

	return controllerBuilder.
		WithOptions(asControllerOptions(opts)).
		Complete(r)
}

// +kubebuilder:rbac:groups=data.nais.io,resources=postgres,verbs=get;list;watch
