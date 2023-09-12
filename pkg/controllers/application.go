package controllers

import (
	"context"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager, opts ...func(*controller.Options)) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1alpha1.Application{}).
		WithOptions(options(opts)).
		Complete(r)
}
