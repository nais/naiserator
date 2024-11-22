package generators

import (
	"context"
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/frontend"
	"github.com/nais/naiserator/pkg/resourcecreator/login"
	"github.com/nais/naiserator/pkg/resourcecreator/observability"
	"github.com/nais/naiserator/pkg/resourcecreator/texas"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/fqdnpolicy"
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
	"github.com/nais/naiserator/pkg/synchronizer"
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

	// Disallow creating application resources if there is a Naisjob with the same name.
	job := &nais_io_v1.Naisjob{}
	err = kube.Get(ctx, key, job)
	if err == nil {
		return nil, fmt.Errorf("cannot create an Application with name '%s' because a Naisjob with that name exists", source.GetName())
	}

	o.NumReplicas = numReplicas(deploy, app.GetReplicas().Min, app.GetReplicas().Max)

	// Retrieve current namespace to check for labels and annotations
	namespaceKey := client.ObjectKey{Name: source.GetNamespace()}
	namespace := &corev1.Namespace{}
	err = kube.Get(ctx, namespaceKey, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing namespace: %s", err)
	}

	// Check if the application is allowed to redirect to another host
	if err = ingress.RedirectAllowed(ctx, app, kube); err != nil {
		return nil, err
	}

	// Auto-detect Google Team Project ID
	o.GoogleTeamProjectID = namespace.Annotations["cnrm.cloud.google.com/project-id"]

	gcpSpec := source.GetGCP()
	if gcpSpec != nil && len(gcpSpec.SqlInstances) == 1 && len(o.GetGoogleTeamProjectID()) > 0 && o.Config.Features.SqlInstanceInSharedVpc {
		instanceName := source.GetName()
		if len(gcpSpec.SqlInstances[0].Name) > 0 {
			instanceName = gcpSpec.SqlInstances[0].Name
		}
		sqlInstanceKey := client.ObjectKey{
			Name:      instanceName,
			Namespace: source.GetNamespace(),
		}
		err = prepareSqlInstance(ctx, kube, sqlInstanceKey, o)
		if err != nil {
			return nil, err
		}
	}

	// Create Linkerd resources only if feature is enabled and namespace is Linkerd-enabled
	if g.Config.Features.Linkerd && namespace.Annotations["linkerd.io/inject"] == "enabled" {
		o.Linkerd = true
	}

	o.Team = app.GetNamespace()

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

	ast := resource.NewAst()

	// This should be first so that other generators can manipulate environment variables without being overwritten.
	pod.CreateContainerEnvVars(app, ast, cfg)

	service.Create(app, ast, cfg)
	serviceaccount.Create(app, ast, cfg)
	horizontalpodautoscaler.Create(app, ast)
	networkpolicy.Create(app, ast, cfg)
	fqdnpolicy.Create(app, ast, cfg)

	err = frontend.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = observability.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = ingress.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = leaderelection.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	azureadapplication, err := azure.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	idportenclient, err := idporten.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = login.Create(app, ast, cfg)
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

	maskinportenclient, err := maskinporten.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	certificateauthority.Create(app, ast, cfg)
	securelogs.Create(app, ast, cfg)
	poddisruptionbudget.Create(app, ast)

	tokenxclient := jwker.Create(app, ast, cfg)

	err = aiven.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = vault.Create(app, ast, cfg)
	if err != nil {
		return nil, err
	}

	// TODO: figure out a better way to provide secret names to Texas
	err = texas.Create(app, ast, cfg, azureadapplication, idportenclient, maskinportenclient, tokenxclient)
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
