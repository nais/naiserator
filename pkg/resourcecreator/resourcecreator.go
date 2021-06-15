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
	"github.com/nais/naiserator/pkg/resourcecreator/kafka"
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
)

// CreateApplication takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func CreateApplication(app *nais_io_v1alpha1.Application, resourceOptions resource.Options) (resource.Operations, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ast := resource.NewAst()

	service.Create(app, ast, *app.Spec.Service)
	serviceaccount.Create(app, ast, resourceOptions)
	horizontalpodautoscaler.Create(app, ast, *app.Spec.Replicas)
	networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
	err := ingress.Create(app, ast, resourceOptions, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
	if err != nil {
		return nil, err
	}
	leaderelection.Create(app, ast, app.Spec.LeaderElection)
	err = azure.Create(app, ast, resourceOptions, *app.Spec.Azure, app.Spec.Ingresses, *app.Spec.AccessPolicy)
	if err != nil {
		return nil, err
	}
	err = idporten.Create(app, ast, resourceOptions, app.Spec.IDPorten, app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}
	kafka.Create(app, ast, resourceOptions, app.Spec.Kafka)
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
	maskinporten.Create(app, ast, resourceOptions, app.Spec.Maskinporten)
	poddisruptionbudget.Create(app, ast, *app.Spec.Replicas)
	jwker.Create(app, ast, resourceOptions, *app.Spec.TokenX, app.Spec.AccessPolicy)
	aiven.Elastic(ast, app.Spec.Elastic)
	linkerd.Create(ast, resourceOptions)

	err = vault.Create(app, ast, resourceOptions, app.Spec.Vault)
	if err != nil {
		return nil, err
	}

	pod.CreateAppContainer(app, ast, resourceOptions)

	err = deployment.Create(app, ast, resourceOptions)
	if err != nil {
		return nil, err
	}

	return ast.Operations, nil
}

// CreateNaisjob takes an Naisjob resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func CreateNaisjob(naisjob *nais_io_v1.Naisjob, resourceOptions resource.Options) (resource.Operations, error) {
	team, ok := naisjob.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ast := resource.NewAst()

	serviceaccount.Create(naisjob, ast, resourceOptions)
	networkpolicy.Create(naisjob, ast, resourceOptions, *naisjob.Spec.AccessPolicy, []nais_io_v1.Ingress{}, false)
	err := azure.Create(naisjob, ast, resourceOptions, *naisjob.Spec.Azure, []nais_io_v1.Ingress{}, *naisjob.Spec.AccessPolicy)
	if err != nil {
		return nil, err
	}
	kafka.Create(naisjob, ast, resourceOptions, naisjob.Spec.Kafka)
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
	maskinporten.Create(naisjob, ast, resourceOptions, naisjob.Spec.Maskinporten)
	aiven.Elastic(ast, naisjob.Spec.Elastic)
	linkerd.Create(ast, resourceOptions)

	err = vault.Create(naisjob, ast, resourceOptions, naisjob.Spec.Vault)
	if err != nil {
		return nil, err
	}

	pod.CreateNaisjobContainer(naisjob, ast, resourceOptions)

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
