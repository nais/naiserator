package vault

import (
	"fmt"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
)

func Create(app *nais_io_v1alpha1.Application, resourceOptions resource.Options, deployment *appsv1.Deployment) error {
	if resourceOptions.VaultEnabled && app.Spec.Vault.Enabled {
		creator, err := NewVaultContainerCreator(*app, resourceOptions)
		if err != nil {
			return fmt.Errorf("while creating Vault creator: %s", err)
		}
		podSpec := &deployment.Spec.Template.Spec

		podSpec, err = creator.AddVaultContainer(podSpec)
		if err != nil {
			return fmt.Errorf("while creating Vault container: %s", err)
		}
		deployment.Spec.Template.Spec = *podSpec
	}

	return nil
}
