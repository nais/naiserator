package resourcecreator_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/deepcomp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	defaultExclude = []string{".apiVersion", ".kind", ".metadata.name"}
)

type SubTest struct {
	ApiVersion string
	Kind       string
	Name       string
	Operation  string
	Match      []Match
}

type Match struct {
	Type     deepcomp.MatchType
	Name     string
	Exclude  []string // list of keys
	Resource interface{}
}

type yamlTestCase struct {
	Config          testCaseConfig
	ResourceOptions resourcecreator.ResourceOptions
	Error           *string
	Input           nais.Application
	Tests           []SubTest
}

func (m meta) String() string {
	return fmt.Sprintf("%s %s/%s %s", m.Operation, m.Resource.ApiVersion, m.Resource.Kind, m.Resource.Metadata.Name)
}

func (m SubTest) String() string {
	return fmt.Sprintf("operation=%s apiVersion=%s kind=%s name=%s", m.Operation, m.ApiVersion, m.Kind, m.Name)
}

func yamlSubtestMatchesResource(resource meta, test SubTest) bool {
	switch {
	case len(test.Name) > 0 && test.Name != resource.Resource.Metadata.Name:
	case len(test.Kind) > 0 && test.Kind != resource.Resource.Kind:
	case len(test.ApiVersion) > 0 && test.ApiVersion != resource.Resource.ApiVersion:
	case len(test.Operation) > 0 && test.Operation != resource.Operation:
	default:
		return true
	}
	return false
}

func resourcemeta(resource interface{}) meta {
	ym := meta{}
	raw, _ := json.Marshal(resource)
	_ = json.Unmarshal(raw, &ym)
	return ym
}

func rawResource(resource runtime.Object) interface{} {
	r := new(interface{})
	raw, _ := yaml.Marshal(resource)
	_ = yaml.Unmarshal(raw, r)
	return r
}

func filter(diffset deepcomp.Diffset, deny func(diff deepcomp.Diff) bool) deepcomp.Diffset {
	diffs := make(deepcomp.Diffset, 0, len(diffset))
	for _, diff := range diffset {
		if !deny(diff) {
			diffs = append(diffs, diff)
		}
	}
	return diffs
}

func yamlRunner(t *testing.T, resources resourcecreator.ResourceOperations, test SubTest) {
	matched := false

	for _, resource := range resources {
		rm := resourcemeta(resource)

		if !yamlSubtestMatchesResource(rm, test) {
			continue
		}
		matched = true

		raw := rawResource(resource.Resource)
		diffs := make(deepcomp.Diffset, 0)

		// retrieve all failure cases
		for _, match := range test.Match {

			// filter out all cases in the exclusion list
			callback := func(diff deepcomp.Diff) bool {
				for _, path := range append(match.Exclude, defaultExclude...) {
					if path == diff.Path {
						return true
					}
				}
				return false
			}

			t.Logf("Assert '%s' against '%s'", match.Name, rm)

			diffs = append(diffs, filter(deepcomp.Compare(match.Type, &match.Resource, raw), callback)...)
		}

		// anything left is an error.
		if len(diffs) > 0 {
			t.Log(diffs.String())
			t.Fail()
		}
	}

	if !matched {
		t.Logf("No resources matching criteria '%s'", test)
		t.Fail()
	}
}

func yamlSubTest(t *testing.T, path string) {
	fixture := fileReader(path)
	data, err := ioutil.ReadAll(fixture)
	if err != nil {
		t.Errorf("unable to read test data: %s", err)
		t.Fail()
		return
	}

	test := yamlTestCase{}
	err = yaml.Unmarshal(data, &test)
	if err != nil {
		t.Errorf("unable to parse unmarshal test data: %s", err)
		t.Fail()
		return
	}

	if test.Config.VaultEnabled {
		viper.Set("features.vault", true)
		viper.Set("vault.address", "https://vault.adeo.no")
		viper.Set("vault.kv-path", "/kv/preprod/fss")
		viper.Set("vault.auth-path", "auth/kubernetes/preprod/fss/login")
		viper.Set("vault.init-container-image", "navikt/vault-sidekick:v0.3.10-d122b16")
	}

	err = nais.ApplyDefaults(&test.Input)
	if err != nil {
		t.Errorf("apply default values to Application object: %s", err)
		t.Fail()
		return
	}

	resources, err := resourcecreator.Create(&test.Input, test.ResourceOptions)
	if test.Error != nil {
		assert.EqualError(t, err, *test.Error)
		return
	}

	assert.NoError(t, err)

	for _, subtest := range test.Tests {
		yamlRunner(t, resources, subtest)
	}
}

func TestNewGoldenFile(t *testing.T) {
	files, err := ioutil.ReadDir(testDataDirectory)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	viper.Set(config.ClusterName, "test-cluster")
	viper.Set(config.ProxyAddress, "http://foo.bar:5224")
	viper.Set(config.ProxyExclude, []string{"foo", "bar", "baz"})

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		path := filepath.Join(testDataDirectory, name)
		t.Run(name, func(t *testing.T) {
			yamlSubTest(t, path)
		})
	}
}
