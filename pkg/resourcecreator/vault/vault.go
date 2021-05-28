package vault

import (
	"fmt"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(objectMeta v1.ObjectMeta, resourceOptions resource.Options, deployment *appsv1.Deployment, naisVault *nais_io_v1alpha1.Vault) error {
	if resourceOptions.VaultEnabled && naisVault.Enabled {
		creator, err := NewVaultContainerCreator(objectMeta, resourceOptions, naisVault)
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
