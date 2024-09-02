package idporten

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/wonderwall"
)

const (
	wonderwallSecretName  = "wonderwall-idporten-config"
	idportenSsoSecretName = "idporten-sso"
)

type Source interface {
	resource.Source
	wonderwall.Source
	GetIDPorten() *nais_io_v1.IDPorten
	GetIngress() []nais_io_v1.Ingress
}

type Config interface {
	wonderwall.Config
	IsWonderwallEnabled() bool
	IsIDPortenEnabled() bool
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	idporten := source.GetIDPorten()
	if idporten == nil || !idporten.Enabled {
		return nil
	}

	if !cfg.IsIDPortenEnabled() {
		return fmt.Errorf("NAISERATOR-9643: idporten is not available in this cluster")
	}

	// TODO - automatically enable sidecar if just idporten is enabled when the grace period for migration ends.
	if idporten.Sidecar == nil || !idporten.Sidecar.Enabled {
		return fmt.Errorf("NAISERATOR-2052: idporten sidecar must be enabled when idporten is enabled")
	}

	if !cfg.IsWonderwallEnabled() {
		return fmt.Errorf("NAISERATOR-7581: idporten sidecar is not enabled for this cluster")
	}

	ingresses := source.GetIngress()
	if len(ingresses) == 0 {
		return fmt.Errorf("NAISERATOR-7816: idporten requires at least 1 ingress")
	}

	ast.Labels["idporten"] = "enabled"
	pod.WithAdditionalSecret(ast, idportenSsoSecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, idportenSsoSecretName)

	return wonderwall.Create(source, ast, cfg, wonderwall.Configuration{
		ACRValues:             idporten.Sidecar.Level,
		AutoLogin:             idporten.Sidecar.AutoLogin,
		AutoLoginIgnorePaths:  idporten.Sidecar.AutoLoginIgnorePaths,
		Ingresses:             ingresses,
		NeedsEncryptionSecret: false,
		Provider:              "idporten",
		SecretNames:           []string{idportenSsoSecretName, wonderwallSecretName},
		Resources:             idporten.Sidecar.Resources,
		UILocales:             idporten.Sidecar.Locale,
	})
}
