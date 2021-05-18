// package resourcecreator converts the Kubernetes custom resource definition
// `nais.io.Applications` into standard Kubernetes resources such as Deployment,
// Service, Ingress, and so forth.

package resourcecreator

import (
	"fmt"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
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
	"github.com/nais/naiserator/pkg/resourcecreator/poddisruptionbudget"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/service"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
)

// Create takes an Application resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options) (resource.Operations, error) {
	team, ok := app.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ops := resource.Operations{}
	service.Create(app, &ops)
	serviceaccount.Create(app, resourceOptions, &ops)
	horizontalpodautoscaler.Create(app, &ops)
	dplt, err := deployment.Create(app, resourceOptions, &ops)
	if err != nil {
		return nil, fmt.Errorf("while creating deployment: %s", err)
	}
	err = azure.Create(app, resourceOptions, dplt, &ops)
	if err != nil {
		return nil, err
	}
	err = idporten.Create(app, resourceOptions, dplt, &ops)
	if err != nil {
		return nil, err
	}
	err = kafka.Create(app, resourceOptions, dplt)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(app, resourceOptions, dplt, &ops)
	if err != nil {
		return nil, err
	}
	maskinporten.Create(app, resourceOptions, dplt, &ops)
	poddisruptionbudget.Create(app, &ops)
	jwker.Create(app, resourceOptions, dplt, &ops)
	leaderelection.Create(app, dplt, &ops)
	aiven.Elastic(app, dplt)
	linkerd.Create(resourceOptions, dplt)
	networkpolicy.Create(app, resourceOptions, &ops)
	err = ingress.Create(app, resourceOptions, &ops)
	if err != nil {
		return nil, fmt.Errorf("while creating ingress: %s", err)
	}

	return ops, nil
}
