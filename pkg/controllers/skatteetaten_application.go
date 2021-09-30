package controllers

import (
	"context"
	"fmt"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/authorization_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/image_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/network_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/postgres"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/service_entry"
	"github.com/nais/naiserator/pkg/skatteetaten_generator/virtual_service"
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
	return r.Synchronizer.Reconcile(ctx, req, &skatteetaten_no_v1alpha1.Application{}, CreateSkatteetatenApplication)
}

func (r *SkatteetatenApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&skatteetaten_no_v1alpha1.Application{}).
		Complete(r)
}




//Hvor bør denne ligge? Vi bør vel ikke legge opp til en abstraksjon hvor vi må endre i de samme filene?
func CreateSkatteetatenApplication(source resource.Source,  resourceOptions resource.Options) (resource.Operations, error) {
	app, ok := source.(*skatteetaten_no_v1alpha1.Application)
	if !ok {
		return nil, fmt.Errorf("BUG: CreateApplication only accepts skatteetaten_no_v1alpha1.Application objects, fix your caller")
	}

	ast := resource.NewAst()

	service.Create(app, ast, resourceOptions)
	serviceaccount.Create(app, ast, resourceOptions)
	horizontalpodautoscaler.Create(app, ast)

	if !app.Spec.UnsecureDebugDisableAllAccessPolicies {
		network_policy.Create(app, ast)
		authorization_policy.Create(app, ast)
	}

	service_entry.Create(app, ast)
	virtual_service.Create(app, ast)
	poddisruptionbudget.Create(app, ast)


	// TODO: Denne er i et annet ns så kan ikke ha owner reference, hvordan får vi slettet ting da?
	err := image_policy.Create(app, ast)
	if err != nil {
		return nil, err
	}

	if app.Spec.Azure != nil {
		postgres.Create(app, ast)
	}
	err = pod.CreateAppContainer(app, ast, resourceOptions)
	if err != nil {
		return nil, err
	}

	err = deployment.Create(app, ast, resourceOptions)
	if err != nil {
		return nil, err
	}

	return ast.Operations, nil
}