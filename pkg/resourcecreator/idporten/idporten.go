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

func Create(source Source, ast *resource.Ast, cfg Config) (*nais_io_v1.IDPortenClient, error) {
	idporten := source.GetIDPorten()
	if idporten == nil || !idporten.Enabled {
		return nil, nil
	}

	if !cfg.IsIDPortenEnabled() {
		return nil, fmt.Errorf("idporten is not available in this cluster")
	}

	// TODO - automatically enable sidecar if just idporten is enabled when the grace period for migration ends.
	if idporten.Sidecar == nil || !idporten.Sidecar.Enabled {
		return nil, fmt.Errorf("idporten sidecar must be enabled when idporten is enabled")
	}

	if !cfg.IsWonderwallEnabled() {
		return nil, fmt.Errorf("idporten sidecar is not enabled for this cluster")
	}

	ingresses := source.GetIngress()
	if len(ingresses) == 0 {
		return nil, fmt.Errorf("idporten requires at least 1 ingress")
	}

	ast.Labels["idporten"] = "enabled"
	pod.WithAdditionalSecret(ast, idportenSsoSecretName, nais_io_v1alpha1.DefaultDigdiratorIDPortenMountPath)
	pod.WithAdditionalEnvFromSecret(ast, idportenSsoSecretName)

	// Construct a fake ID-porten client as we're using a shared client across all applications.
	client := &nais_io_v1.IDPortenClient{
		Spec: nais_io_v1.IDPortenClientSpec{
			SecretName: idportenSsoSecretName,
		},
	}

	return client, wonderwall.Create(source, ast, cfg, wonderwall.Configuration{
		ACRValues:             idporten.Sidecar.Level,
		AutoLogin:             idporten.Sidecar.AutoLogin,
		AutoLoginIgnorePaths:  idporten.Sidecar.AutoLoginIgnorePaths,
		NeedsEncryptionSecret: false,
		Provider:              "idporten",
		SecretNames:           []string{idportenSsoSecretName, wonderwallSecretName},
		Resources:             idporten.Sidecar.Resources,
		UILocales:             idporten.Sidecar.Locale,
	})
}
