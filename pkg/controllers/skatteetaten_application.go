package controllers

import (
	"context"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/synchronizer"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SkatteetatenApplicationReconciler struct {
	synchronizer.Synchronizer
}

func NewSkatteetatenAppReconciler(synchronizer synchronizer.Synchronizer) *SkatteetatenApplicationReconciler {
	return &SkatteetatenApplicationReconciler{Synchronizer: synchronizer}
}

// +kubebuilder:rbac:groups=application.nebula.skatteetaten.no,resources=Applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=application.nebula.skatteetaten.no,resources=Applications/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *SkatteetatenApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.Synchronizer.ReconcileSkatteetatenApplication(req)
}

func (r *SkatteetatenApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&skatteetaten_no_v1alpha1.Application{}).
		Complete(r)
}
