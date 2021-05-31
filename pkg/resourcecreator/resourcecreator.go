// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	deployment "github.com/nais/naiserator/pkg/resourcecreator/deployment"
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

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options) (resource.Operations, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	// TODO: Disse nedenfor kan fjernes hvis vi sender inn source i stedet, da kan man lage de fortløpende
	ast := resource.NewAst()
	service.Create(app, ast, *app.Spec.Service)
	serviceaccount.Create(app, ast, resourceOptions)
	horizontalpodautoscaler.Create(app, ast, *app.Spec.Replicas)
	networkpolicy.Create(app, ast, resourceOptions, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
	err := ingress.Create(app, ast, resourceOptions, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
	if err != nil {
		return nil, fmt.Errorf("while creating ingress: %s", err)
	}

	leaderelection.Create(app, ast, app.Spec.LeaderElection)
	err = azure.Create(app, ast, resourceOptions, ast, *app.Spec.Azure, app.Spec.Ingresses, *app.Spec.AccessPolicy)
	if err != nil {
		return nil, err
	}
	err = idporten.Create(app, ast, resourceOptions, app.Spec.IDPorten, app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}
	err = kafka.Create(app, ast, resourceOptions, app.Spec.Kafka)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(app, ast, resourceOptions, app.Spec.GCP)
	if err != nil {
		return nil, err
	}
	err = proxyopts.Create(resourceOptions, deploy, app.Spec.WebProxy)
	if err != nil {
		return nil, err
	}
	certificateauthority.Create(deploy, app.Spec.SkipCaBundle)
	securelogs.Create(resourceOptions, deploy, app.Spec.SecureLogs)
	maskinporten.Create(objectMeta, resourceOptions, deploy, ast, app.Spec.Maskinporten)
	poddisruptionbudget.Create(objectMeta, ast, *app.Spec.Replicas)
	jwker.Create(objectMeta, resourceOptions, deploy, ast, *app.Spec.TokenX, app.Spec.AccessPolicy)
	aiven.Elastic(deploy, app.Spec.Elastic)
	linkerd.Create(resourceOptions, deploy)
	vault.Create(objectMeta, resourceOptions, deploy, app.Spec.Vault)
	pod.CreateAppContainer(app, ast, resourceOptions) // skulle denne vært i deployment-kallet?
	err = deployment.Create(app, ast, resourceOptions)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}

	return ops, nil
}
