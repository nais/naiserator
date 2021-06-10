package controllers

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/synchronizer"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ApplicationReconciler struct {
	synchronizer.Synchronizer
}

func NewAppReconciler(synchronizer synchronizer.Synchronizer) *ApplicationReconciler {
	return &ApplicationReconciler{Synchronizer: synchronizer}
}

// +kubebuilder:rbac:groups=nais.io,resources=Applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=Applications/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Synchronizer.ReconcileApplication(req)
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1alpha1.Application{}).
		Complete(r)
}
