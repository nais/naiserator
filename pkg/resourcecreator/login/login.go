package login

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/wonderwall"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	// secret name for global configuration of the login proxy, e.g. redis properties
	globalSecretName = "login-global-config"
)

type Source interface {
	resource.Source
	wonderwall.Source
}

type Config interface {
	wonderwall.Config
	IsLoginProxyEnabled() bool
	IsWonderwallEnabled() bool
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	login := source.GetLogin()
	if login == nil {
		return nil
	}

	if !cfg.IsLoginProxyEnabled() {
		return fmt.Errorf("login proxy is not available in this cluster")
	}

	if !cfg.IsWonderwallEnabled() {
		return fmt.Errorf("login proxy is enabled but wonderwall is not")
	}

	ingresses := source.GetIngress()
	if len(ingresses) == 0 {
		return fmt.Errorf("login proxy requires at least 1 ingress")
	}

	applicationSecretName, err := applicationSecretName(source)
	if err != nil {
		return err
	}

	ast.Labels["login-proxy"] = "enabled"

	return wonderwall.Create(source, ast, cfg, wonderwall.Configuration{
		AutoLogin: login.Enforce != nil && login.Enforce.Enabled,
		AutoLoginIgnorePaths: func() []nais_io_v1.WonderwallIgnorePaths {
			if login.Enforce == nil {
				return nil
			}
			return login.Enforce.ExcludePaths
		}(),
		NeedsEncryptionSecret: true,
		Provider:              login.Provider,
		SecretNames:           []string{applicationSecretName, globalSecretName},
	})
}

// secret name for application-specific configuration of the login proxy, using the following scheme:
// `login-config-<application-name>`
// assumes that application name is less than 253-len("login-config")
func applicationSecretName(source Source) (string, error) {
	name := fmt.Sprintf("login-config-%s", source.GetName())

	if len(name) > validation.DNS1123SubdomainMaxLength {
		return "", fmt.Errorf("application name too long: %s", source.GetName())
	}

	return name, nil
}
