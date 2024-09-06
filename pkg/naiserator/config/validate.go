package config

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
)

func (v Vault) Validate() error {
	var result = &multierror.Error{}

	if len(v.Address) == 0 {
		multierror.Append(result, fmt.Errorf("NAISERATOR-7612: vault address not found in environment"))
	}

	if len(v.InitContainerImage) == 0 {
		multierror.Append(result, fmt.Errorf("NAISERATOR-8218: vault init container image not found in environment"))
	}

	if len(v.AuthPath) == 0 {
		multierror.Append(result, fmt.Errorf("NAISERATOR-9099: vault auth path not found in environment"))
	}

	if len(v.KeyValuePath) == 0 {
		multierror.Append(result, fmt.Errorf("NAISERATOR-3997: vault kv path not specified"))
	}

	return result.ErrorOrNil()
}
