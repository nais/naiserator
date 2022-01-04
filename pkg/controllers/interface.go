package controllers

import (
	"context"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Interface interface {
	Reconcile(ctx context.Context, req ctrl.Request, app resource.Source) (ctrl.Result, error)
}
