package generator

import (
	"fmt"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/google/gcp"
	"github.com/nais/naiserator/pkg/resourcecreator/horizontalpodautoscaler"
	"github.com/nais/naiserator/pkg/resourcecreator/idporten"
	"github.com/nais/naiserator/pkg/resourcecreator/ingress"
	"github.com/nais/naiserator/pkg/resourcecreator/jwker"
	"github.com/nais/naiserator/pkg/resourcecreator/leaderelection"
	"github.com/nais/naiserator/pkg/resourcecreator/linkerd"
	"github.com/nais/naiserator/pkg/resourcecreator/maskinporten"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/nais/naiserator/pkg/resourcecreator/wonderwall"
	"github.com/nais/naiserator/pkg/synchronizer"
)

type Application struct {
	Config config.Config
}

var _ synchronizer.Generator = &Application{}

// CreateApplication takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func (g *Application) Generate(source resource.Source, resourceOptions resource.Options) (resource.Operations, error) {
	var err error

	app, ok := source.(*nais_io_v1alpha1.Application)
	if !ok {
		return nil, fmt.Errorf("BUG: CreateApplication only accepts nais_io_v1alpha1.Application objects, fix your caller")
	}

	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	if app.Spec.GCP != nil && len(resourceOptions.GoogleTeamProjectId) == 0 {
		// We're not currently in a team namespace with corresponding GCP team project
		return nil, fmt.Errorf("GCP resources requested, but no team project ID annotation set on namespace %s (not running on GCP?)", app.GetNamespace())
	}

	resourceOptions.WonderwallEnabled, err = wonderwall.ShouldEnable(app, resourceOptions)
	if err != nil {
		return nil, err
	}

	ast := resource.NewAst()

	service.Create(app, ast, resourceOptions)
	serviceaccount.Create(app, ast, resourceOptions)

	if !app.Spec.Replicas.DisableAutoScaling && app.Spec.Replicas.Min != app.Spec.Replicas.Max {
		horizontalpodautoscaler.Create(app, ast)
	}

	networkpolicy.Create(app, ast, &g.Config, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
	err = ingress.Create(app, ast, &g.Config, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
	if err != nil {
		return nil, err
	}
	leaderelection.Create(app, ast, app.Spec.LeaderElection)
	err = azure.Create(app, ast, resourceOptions)
	if err != nil {
		return nil, err
	}
	err = idporten.Create(app, ast, resourceOptions)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(app, ast, resourceOptions, app.Spec.GCP, &g.Config)
	if err != nil {
		return nil, err
	}
	err = proxyopts.Create(ast, resourceOptions, app.Spec.WebProxy)
	if err != nil {
		return nil, err
	}
	certificateauthority.Create(ast, app.Spec.SkipCaBundle)
	securelogs.Create(ast, resourceOptions, app.Spec.SecureLogs)
	err = maskinporten.Create(app, ast, resourceOptions, app.Spec.Maskinporten)
	if err != nil {
		return nil, err
	}
	poddisruptionbudget.Create(app, ast)

	jwker.Create(app, ast, resourceOptions, *app.Spec.TokenX, app.Spec.AccessPolicy)
	linkerd.Create(ast, resourceOptions)

	aivenSpecs := aiven.Specs{
		Kafka:   app.Spec.Kafka,
		Elastic: app.Spec.Elastic,
		Influx:  app.Spec.Influx,
	}
	err = aiven.Create(app, ast, &g.Config, aivenSpecs)
	if err != nil {
		return nil, err
	}

	err = vault.Create(app, ast, resourceOptions, app.Spec.Vault)
	if err != nil {
		return nil, err
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
