package generators

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	"github.com/nais/naiserator/pkg/resourcecreator/batch"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/google/gcp"
	"github.com/nais/naiserator/pkg/resourcecreator/linkerd"
	"github.com/nais/naiserator/pkg/resourcecreator/maskinporten"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/nais/naiserator/pkg/synchronizer"
)

type Naisjob struct {
	Config config.Config
}

var _ synchronizer.Generator = &Naisjob{}

// CreateNaisjob takes an Naisjob resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func (g *Naisjob) Generate(source resource.Source, resourceOptions resource.Options) (resource.Operations, error) {
	naisjob, ok := source.(*nais_io_v1.Naisjob)
	if !ok {
		return nil, fmt.Errorf("BUG: generator only accepts nais_io_v1.Naisjob objects, fix your caller")
	}

	team, ok := naisjob.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ast := resource.NewAst()

	serviceaccount.Create(naisjob, ast, resourceOptions)
	networkpolicy.Create(naisjob, ast, &g.Config, *naisjob.Spec.AccessPolicy, []nais_io_v1.Ingress{}, false)
	err := azure.Create(naisjob, ast, resourceOptions)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(naisjob, ast, resourceOptions, naisjob.Spec.GCP, &g.Config)
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

	aivenSpecs := aiven.Specs{
		Kafka:   naisjob.Spec.Kafka,
		Elastic: naisjob.Spec.Elastic,
		Influx:  naisjob.Spec.Influx,
	}
	err = aiven.Create(naisjob, ast, &g.Config, aivenSpecs)
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
