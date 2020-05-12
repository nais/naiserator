package resourcecreator_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/deepcomp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
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
	Resource interface{}
}

type yamlTestCase struct {
	Config          testCaseConfig
	ResourceOptions resourcecreator.ResourceOptions
	Error           *string
	Input           nais.Application
	Tests           []SubTest
}

type yamlmeta struct {
	Operation string
	Resource  struct {
		TypeMeta struct {
			ApiVersion string
			Kind       string
		}
		ObjectMeta struct {
			Name string
		}
	}
}

func (m meta) String() string {
	return fmt.Sprintf("%s %s/%s %s", m.Operation, m.Resource.ApiVersion, m.Resource.Kind, m.Resource.Metadata.Name)
}

func (s SubTest) String() string {
	return "TODO"
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
	raw, _ := json.Marshal(resource)
	_ = json.Unmarshal(raw, &r)
	return r
}

func yamlRunner(t *testing.T, resources resourcecreator.ResourceOperations, test SubTest) {
	var err error

	for _, resource := range resources {
		rm := resourcemeta(resource)

		if !yamlSubtestMatchesResource(rm, test) {
			continue
		}
		t.Logf("testing resource %s against %s", rm, test)

		raw := rawResource(resource.Resource)
		for _, match := range test.Match {
			deepcomp.Compare(match.Type, match.Resource, raw)
		}

		if err != nil {
			t.Error(err)
		}
	}
}

func yamlSubTest(t *testing.T, path string) {
	fixture := fileReader(path)
	data, err := ioutil.ReadAll(fixture)
	if err != nil {
		t.Errorf("unable to read test data: %s", err)
		t.Fail()
	}

	test := yamlTestCase{}
	err = yaml.Unmarshal(data, &test)
	if err != nil {
		t.Errorf("unable to parse unmarshal test data: %s", err)
		t.Fail()
	}

	err = nais.ApplyDefaults(&test.Input)
	if err != nil {
		t.Errorf("apply default values to Application object: %s", err)
		t.Fail()
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
