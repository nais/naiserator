package skatteetaten_resourcecreator_test

import (
	"fmt"
	"testing"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/skatteetaten_resourcecreator"
	"github.com/nais/naiserator/pkg/test/goldenfile"

	"github.com/ghodss/yaml"
)

const (
	skatteeteatenApplicationTestDataDirectory = "testdata"
)

type skatteetatenApplicationTestCase struct {
	Input skatteetaten_no_v1alpha1.Application
}


func TestSkatteetatenApplicationGoldenFile(t *testing.T) {
	goldenfile.Run(t, skatteeteatenApplicationTestDataDirectory, func(input []byte, resourceOptions resource.Options) (resource.Operations, error) {
		test := skatteetatenApplicationTestCase{}
		err := yaml.Unmarshal(input, &test)
		if err != nil {
			return nil, err
		}

		err = test.Input.ApplyDefaults()
		if err != nil {
			return nil, fmt.Errorf("apply default values to Application object: %s", err)
		}

		return skatteetaten_resourcecreator.CreateSkatteetatenApplication(&test.Input, resourceOptions)
	})
}
