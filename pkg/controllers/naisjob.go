package controllers

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/synchronizer"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NaisjobReconciler reconciles a Naisjob object
type NaisjobReconciler struct {
	synchronizer.Synchronizer
}

func NewNaisjobReconciler(synchronizer synchronizer.Synchronizer) *NaisjobReconciler {
	return &NaisjobReconciler{Synchronizer: synchronizer}
}

// +kubebuilder:rbac:groups=nais.io,resources=Naisjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=Naisjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *NaisjobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Synchronizer.ReconcileNaisjob(req)
}

func (r *NaisjobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1.Naisjob{}).
		Complete(r)
}
