package util

import (
	"fmt"
	"net/url"
	"regexp"
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
