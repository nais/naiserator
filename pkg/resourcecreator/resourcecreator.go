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

	objectMeta := app.CreateObjectMeta()
	appNamespaceHash := app.CreateAppNamespaceHash()
	ast := resource.NewAst(app)
	ops := resource.Operations{}
	service.Create(objectMeta, &ops, *app.Spec.Service)
	serviceaccount.Create(objectMeta, resourceOptions, &ops, appNamespaceHash)
	horizontalpodautoscaler.Create(objectMeta, &ops, *app.Spec.Replicas)
	leaderelection.Create(ast, app.Spec.LeaderElection)
	pod.CreateAppContainer(ast, app)
	deploy, err := deployment.Create(objectMeta, resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
		app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
		app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	err = azure.Create(objectMeta, resourceOptions, deploy, &ops, *app.Spec.Azure, app.Spec.Ingresses, *app.Spec.AccessPolicy)
	if err != nil {
		return nil, err
	}
	err = idporten.Create(objectMeta, resourceOptions, deploy, &ops, app.Spec.IDPorten, app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}
	err = kafka.Create(objectMeta, resourceOptions, deploy, app.Spec.Kafka)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(objectMeta, resourceOptions, deploy, &ops, appNamespaceHash, app.Spec.GCP)
	if err != nil {
		return nil, err
	}
	err = proxyopts.Create(resourceOptions, deploy, app.Spec.WebProxy)
	if err != nil {
		return nil, err
	}
	certificateauthority.Create(deploy, app.Spec.SkipCaBundle)
	securelogs.Create(resourceOptions, deploy, app.Spec.SecureLogs)
	maskinporten.Create(objectMeta, resourceOptions, deploy, &ops, app.Spec.Maskinporten)
	poddisruptionbudget.Create(objectMeta, &ops, *app.Spec.Replicas)
	jwker.Create(objectMeta, resourceOptions, deploy, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
	aiven.Elastic(deploy, app.Spec.Elastic)
	linkerd.Create(resourceOptions, deploy)
	networkpolicy.Create(objectMeta, resourceOptions, &ops, *app.Spec.AccessPolicy, app.Spec.Ingresses, app.Spec.LeaderElection)
	err = ingress.Create(objectMeta, resourceOptions, &ops, app.Spec.Ingresses, app.Spec.Liveness.Path, app.Spec.Service.Protocol, app.Annotations)
	if err != nil {
		return nil, fmt.Errorf("while creating ingress: %s", err)
	}
	vault.Create(objectMeta, resourceOptions, deploy, app.Spec.Vault)

	return ops, nil
}
