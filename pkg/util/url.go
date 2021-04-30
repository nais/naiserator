package util

import (
	"fmt"
	"net/url"
	"path"
	"regexp"

	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
)

var kubernetesFQDNValidation = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

func ValidateUrl(u *url.URL) error {
	if len(u.Host) == 0 {
		return fmt.Errorf("URL '%s' is missing a hostname", u)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("URL '%s' does not start with 'https://'", u)
	}

	if !kubernetesFQDNValidation.MatchString(u.Host) {
		return fmt.Errorf("URL '%s' does not match regular expression '%s'", u, kubernetesFQDNValidation.String())
	}

	return nil
}

func AppendPathToIngress(ingress nais_io_v1alpha1.Ingress, joinPath string) string {
	u, _ := url.Parse(string(ingress))
	u.Path = path.Join(u.Path, joinPath)
	return u.String()
}

