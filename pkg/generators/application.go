package generators

import (
	"context"
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
	"github.com/nais/naiserator/pkg/resourcecreator/maskinporten"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/podmonitor"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/nais/naiserator/pkg/resourcecreator/wonderwall"
	"github.com/nais/naiserator/pkg/synchronizer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Application struct {
	Config config.Config
}

var _ synchronizer.Generator = &Application{}

// Prepare a configuration context for further processing.
// This function detects run-time parameters from a live running cluster.
func (g *Application) Prepare(ctx context.Context, source resource.Source, kube client.Client) (interface{}, error) {
	app, ok := source.(*nais_io_v1alpha1.Application)
	if !ok {
		return nil, fmt.Errorf("BUG: this generator accepts only nais_io_v1alpha1.Application objects")
	}

	o := &Options{
		Config: g.Config,
	}

	// Make a query to Kubernetes for this application's previous deployment.
	// The number of replicas is significant, so we need to carry it over to match
	// this next rollout.
	key := client.ObjectKey{
		Name:      source.GetName(),
		Namespace: source.GetNamespace(),
	}
	deploy := &appsv1.Deployment{}
	err := kube.Get(ctx, key, deploy)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing deployment: %s", err)
	}

	o.NumReplicas = numReplicas(deploy, app.GetReplicas().Min, app.GetReplicas().Max)

	// Retrieve current namespace to check for labels and annotations
	key = client.ObjectKey{Name: source.GetNamespace()}
	namespace := &corev1.Namespace{}
	err = kube.Get(ctx, key, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing namespace: %s", err)
	}

	// Auto-detect Google Team Project ID
	o.GoogleTeamProjectID = namespace.Annotations["cnrm.cloud.google.com/project-id"]

	// Create Linkerd resources only if feature is enabled and namespace is Linkerd-enabled
	if g.Config.Features.Linkerd && namespace.Annotations["linkerd.io/inject"] == "enabled" {
		o.Linkerd = true
	}

	o.WonderwallEnabled, err = wonderwall.ShouldEnable(app, o)
	if err != nil {
		return nil, err
	}

	o.Team = app.Labels["team"]

	return o, nil
}

// Generate transforms an Application resource into a set of Kubernetes resources,
// along with information about what to do with these resources, i.e. CreateOrUpdate, etc.
func (g *Application) Generate(source resource.Source, config interface{}) (resource.Operations, error) {
	var err error

	app, ok := source.(*nais_io_v1alpha1.Application)
	if !ok {
		return nil, fmt.Errorf("BUG: CreateApplication only accepts nais_io_v1alpha1.Application objects; fix your code")
	}

	cfg, ok := config.(*Options)
	if !ok {
		return nil, fmt.Errorf("BUG: Application generator called without correct configuration object; fix your code")
	}

	if len(cfg.GetTeam()) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ast := resource.NewAst()

	service.Create(app, ast, cfg)
	serviceaccount.Create(app, ast, cfg)
	horizontalpodautoscaler.Create(app, ast)
	networkpolicy.Create(app, ast, cfg)

	err = ingress.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = leaderelection.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = azure.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = idporten.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = gcp.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = proxyopts.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	certificateauthority.Create(app, ast, cfg)
	securelogs.Create(app, ast, cfg)
	err = maskinporten.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}
	poddisruptionbudget.Create(app, ast)

	jwker.Create(app, ast, cfg)

	err = aiven.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = vault.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = pod.CreateAppContainer(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = deployment.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	podmonitor.Create(app, ast, cfg)

	return ast.Operations, nil
}
