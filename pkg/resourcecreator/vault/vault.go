package vault

import (
	"fmt"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment) error {
	if resourceOptions.VaultEnabled && app.Spec.Vault.Enabled {
		podSpec := &deployment.Spec.Template.Spec
		podSpec, err := vaultSidecar(app, podSpec)
		if err != nil {
			return err
		}
		deployment.Spec.Template.Spec = *podSpec
	}

	return nil
}

func vaultSidecar(app *nais_io_v1alpha1.Application, podSpec *v1.PodSpec) (*v1.PodSpec, error) {
	creator, err := NewVaultContainerCreator(*app)
	if err != nil {
		return nil, fmt.Errorf("while creating Vault container: %s", err)
	}
	return creator.AddVaultContainer(podSpec)
}
