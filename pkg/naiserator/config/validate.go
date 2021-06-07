package config

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
)

func (v Vault) Validate() error {
	var result = &multierror.Error{}

	if len(v.Address) == 0 {
		multierror.Append(result, fmt.Errorf("vault address not found in environment"))
	}

	if len(v.InitContainerImage) == 0 {
		multierror.Append(result, fmt.Errorf("vault init container image not found in environment"))
	}

	if len(v.AuthPath) == 0 {
		multierror.Append(result, fmt.Errorf("vault auth path not found in environment"))
	}

	if len(v.KeyValuePath) == 0 {
		multierror.Append(result, fmt.Errorf("vault kv path not specified"))
	}

	return result.ErrorOrNil()
}
