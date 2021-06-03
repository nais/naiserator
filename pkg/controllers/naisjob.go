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

// +kubebuilder:rbac:groups=nais.io.nais.io,resources=naisjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais_io_v1.io.nais_io_v1.io,resources=naisjobs/status,verbs=get;update;patch

func (r *NaisjobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Synchronizer.ReconcileNaisjob(req, &nais_io_v1.Naisjob{})
}

func (r *NaisjobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1.Naisjob{}).
		Complete(r)
}
