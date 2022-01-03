package resourcecreator_test

import (
	"fmt"
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/generators"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/goldenfile"

	"github.com/ghodss/yaml"
)

const (
	applicationTestDataDirectory = "testdata"
	naisjobTestDataDirectory     = "testdata/naisjob"
)

type applicationTestCase struct {
	Input nais_io_v1alpha1.Application
}

type naisjobTestCase struct {
	Input nais_io_v1.Naisjob
}

func TestApplicationGoldenFile(t *testing.T) {
	goldenfile.Run(t, applicationTestDataDirectory, func(input []byte, resourceOptions generators.Options, config config.Config) (resource.Operations, error) {
		test := applicationTestCase{}
		err := yaml.Unmarshal(input, &test)
		if err != nil {
			return nil, err
		}

		err = test.Input.ApplyDefaults()
		if err != nil {
			return nil, fmt.Errorf("apply default values to Application object: %s", err)
		}

		gen := &generators.Application{
			Config: config,
		}

		return gen.Generate(&test.Input, resourceOptions)
	})
}

func TestNaisjobGoldenFile(t *testing.T) {
	goldenfile.Run(t, naisjobTestDataDirectory, func(input []byte, resourceOptions generators.Options, config config.Config) (resource.Operations, error) {
		test := naisjobTestCase{}
		err := yaml.Unmarshal(input, &test)
		if err != nil {
			return nil, err
		}

		err = test.Input.ApplyDefaults()
		if err != nil {
			return nil, fmt.Errorf("apply default values to Naisjob object: %s", err)
		}

		gen := &generators.Naisjob{
			Config: config,
		}

		return gen.Generate(&test.Input, resourceOptions)
	})
}
