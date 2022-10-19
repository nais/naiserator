package generators

import (
	"context"
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/aiven"
	"github.com/nais/naiserator/pkg/resourcecreator/azure"
	"github.com/nais/naiserator/pkg/resourcecreator/batch"
	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/google/gcp"
	"github.com/nais/naiserator/pkg/resourcecreator/maskinporten"
	"github.com/nais/naiserator/pkg/resourcecreator/networkpolicy"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/securelogs"
	"github.com/nais/naiserator/pkg/resourcecreator/serviceaccount"
	"github.com/nais/naiserator/pkg/resourcecreator/vault"
	"github.com/nais/naiserator/pkg/synchronizer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Naisjob struct {
	Config config.Config
}

var _ synchronizer.Generator = &Naisjob{}

// Generate a configuration context for further processing.
// This function detects run-time parameters from a live running cluster.
func (g *Naisjob) Prepare(ctx context.Context, source resource.Source, kube client.Client) (interface{}, error) {
	job, ok := source.(*nais_io_v1.Naisjob)
	if !ok {
		return nil, fmt.Errorf("BUG: this generator accepts only nais_io_v1.Naisjob objects")
	}

	o := &Options{
		Config: g.Config,
	}

	o.NumReplicas = 1

	// Retrieve current namespace to check for labels and annotations
	key := client.ObjectKey{Name: source.GetNamespace()}
	namespace := &corev1.Namespace{}
	err := kube.Get(ctx, key, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("query existing namespace: %s", err)
	}

	// Auto-detect Google Team Project ID
	o.GoogleTeamProjectID = namespace.Annotations["cnrm.cloud.google.com/project-id"]

	// Create Linkerd resources only if feature is enabled and namespace is Linkerd-enabled
	if g.Config.Features.Linkerd && namespace.Annotations["linkerd.io/inject"] == "enabled" {
		o.Linkerd = true
	}

	o.Team = job.Labels["team"]

	return o, nil
}

// CreateNaisjob takes an Naisjob resource and returns a slice of Kubernetes resources
// along with information about what to do with these resources.
func (g *Naisjob) Generate(source resource.Source, config interface{}) (resource.Operations, error) {
	naisjob, ok := source.(*nais_io_v1.Naisjob)
	if !ok {
		return nil, fmt.Errorf("BUG: generator only accepts nais_io_v1.Naisjob objects, fix your caller")
	}

	cfg, ok := config.(*Options)
	if !ok {
		return nil, fmt.Errorf("BUG: Application generator called without correct configuration object; fix your code")
	}

	team, ok := naisjob.Labels["team"]
	if !ok || len(team) == 0 {
		return nil, fmt.Errorf("the 'team' label needs to be set in the application metadata")
	}

	ast := resource.NewAst()

	serviceaccount.Create(naisjob, ast, cfg)
	networkpolicy.Create(naisjob, ast, cfg)
	err := azure.Create(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}
	err = gcp.Create(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}
	err = proxyopts.Create(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}
	certificateauthority.Create(naisjob, ast, cfg)
	securelogs.Create(naisjob, ast, cfg)
	err = maskinporten.Create(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = aiven.Create(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = vault.Create(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}

	err = pod.CreateNaisjobContainer(naisjob, ast, cfg)
	if err != nil {
		return nil, err
	}

	if naisjob.Spec.Schedule == "" {
		if err := batch.CreateJob(naisjob, ast, cfg); err != nil {
			return nil, err
		}
	} else {
		if err := batch.CreateCronJob(naisjob, ast, cfg); err != nil {
			return nil, err
		}
	}

	return ast.Operations, nil
}
