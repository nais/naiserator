package skatteetaten_resourcecreator

import (
	"fmt"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/azure/cosmosdb"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/azure/postgres"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/azure/storageaccount"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/image_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/istio/authorization_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/istio/network_policy"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/istio/service_entry"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator/istio/virtual_service"
)

func CreateSkatteetatenApplication(source resource.Source, resourceOptions resource.Options) (resource.Operations, error) {
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
	virtual_service.Create(app, ast, resourceOptions)
	poddisruptionbudget.Create(app, ast)

	err := image_policy.Create(app, ast)
	if err != nil {
		return nil, err
	}

	if app.Spec.Azure != nil {
		postgres.Create(app, ast)
		storageaccount.Create(app, ast)
		cosmosdb.Create(app, ast)
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
