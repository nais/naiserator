package controllers

import (
	"context"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type NaisjobReconciler struct {
	synchronizer Interface
}

func NewNaisjobReconciler(synchronizer Interface) *NaisjobReconciler {
	return &NaisjobReconciler{
		synchronizer: synchronizer,
	}
}

// +kubebuilder:rbac:groups=nais.io,resources=Naisjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=Naisjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *NaisjobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.synchronizer.Reconcile(ctx, req, &nais_io_v1.Naisjob{})
}

func (r *NaisjobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1.Naisjob{}).
		Complete(r)
}
