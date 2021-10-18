// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"

	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	"github.com/nais/naiserator/pkg/resourcecreator/batch"
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
)

type Generator func(app resource.Source, resourceOptions resource.Options) (resource.Operations, error)

// CreateApplication takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func CreateApplication(source resource.Source, resourceOptions resource.Options) (resource.Operations, error) {
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

	if app.Spec.Replicas.Min != app.Spec.Replicas.Max {
		horizontalpodautoscaler.Create(app, ast)
	}

	networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
	err = ingress.Create(app, ast, resourceOptions, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
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
	err = gcp.Create(app, ast, resourceOptions, app.Spec.GCP)
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

	aivenSpecs := aiven.AivenSpecs{
		Kafka:   app.Spec.Kafka,
		Elastic: app.Spec.Elastic,
		Influx:  app.Spec.Influx,
	}
	err = aiven.Create(app, ast, resourceOptions, aivenSpecs)
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

// CreateNaisjob takes an Naisjob resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func CreateNaisjob(source resource.Source, resourceOptions resource.Options) (resource.Operations, error) {
	naisjob, ok := source.(*nais_io_v1.Naisjob)
	if !ok {
		return nil, fmt.Errorf("BUG: CreateApplication only accepts nais_io_v1alpha1.Application objects, fix your caller")
	}

	team, ok := naisjob.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ast := resource.NewAst()

	serviceaccount.Create(naisjob, ast, resourceOptions)
	networkpolicy.Create(naisjob, ast, resourceOptions, *naisjob.Spec.AccessPolicy, []nais_io_v1.Ingress{}, false)
	err := azure.Create(naisjob, ast, resourceOptions)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(naisjob, ast, resourceOptions, naisjob.Spec.GCP)
	if err != nil {
		return nil, err
	}
	err = proxyopts.Create(ast, resourceOptions, naisjob.Spec.WebProxy)
	if err != nil {
		return nil, err
	}
	certificateauthority.Create(ast, naisjob.Spec.SkipCaBundle)
	securelogs.Create(ast, resourceOptions, naisjob.Spec.SecureLogs)
	err = maskinporten.Create(naisjob, ast, resourceOptions, naisjob.Spec.Maskinporten)
	if err != nil {
		return nil, err
	}

	linkerd.Create(ast, resourceOptions)

	aivenSpecs := aiven.AivenSpecs{
		Kafka:   naisjob.Spec.Kafka,
		Elastic: naisjob.Spec.Elastic,
		Influx:  naisjob.Spec.Influx,
	}
	err = aiven.Create(naisjob, ast, resourceOptions, aivenSpecs)
	if err != nil {
		return nil, err
	}

	err = vault.Create(naisjob, ast, resourceOptions, naisjob.Spec.Vault)
	if err != nil {
		return nil, err
	}

	err = pod.CreateNaisjobContainer(naisjob, ast, resourceOptions)
	if err != nil {
		return nil, err
	}

	if naisjob.Spec.Schedule == "" {
		if err := batch.CreateJob(naisjob, ast, resourceOptions); err != nil {
			return nil, err
		}
	} else {
		if err := batch.CreateCronJob(naisjob, ast, resourceOptions); err != nil {
			return nil, err
		}
	}

	return ast.Operations, nil
}
