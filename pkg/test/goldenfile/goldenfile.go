package goldenfile

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/stretchr/testify/assert"

	"github.com/ghodss/yaml"
	"github.com/nais/naiserator/pkg/test/deepcomp"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	defaultExclude = []string{".apiVersion", ".kind", ".metadata.name"}
)

type testCaseConfig struct {
	Description string
	MatchType   string
}

type meta struct {
	Operation string
	Resource  struct {
		ApiVersion string
		Kind       string
		Metadata   struct {
			Name string
		}
	}
}

func fileReader(file string) io.Reader {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	return f
}

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

type TestCase struct {
	Config          testCaseConfig
	ResourceOptions resource.Options
	Error           *string
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

func yamlRunner(t *testing.T, filename string, resources resource.Operations, test SubTest) {
	matched := false

	for _, rsce := range resources {
		rm := resourcemeta(rsce)

		if !yamlSubtestMatchesResource(rm, test) {
			continue
		}
		matched = true

		raw := rawResource(rsce.Resource)
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

			t.Logf("%s: Assert '%s' against '%s'", filename, match.Name, rm)

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

func yamlSubTest(t *testing.T, path string, createOperations CreateOperationsCallback) {
	fixture := fileReader(path)
	data, err := ioutil.ReadAll(fixture)
	if err != nil {
		t.Errorf("unable to read test data: %s", err)
		t.Fail()
		return
	}

	test := TestCase{}
	err = yaml.Unmarshal(data, &test)
	if err != nil {
		t.Errorf("unable to unmarshal test data: %s", err)
		t.Fail()
		return
	}

	resources, err := createOperations(data, test.ResourceOptions)
	if err != nil {
		if test.Error != nil {
			assert.EqualError(t, err, *test.Error)
			return
		}
		t.Errorf("unable to unmarshal test data input: %s", err)
		t.Fail()
		return
	}

	for _, subtest := range test.Tests {
		yamlRunner(t, filepath.Base(path), resources, subtest)
	}
}

type CreateOperationsCallback func([]byte, resource.Options) (resource.Operations, error)

func Run(t *testing.T, testDataDirectory string, createOperations CreateOperationsCallback) {
	files, err := ioutil.ReadDir(testDataDirectory)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

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
			yamlSubTest(t, path, createOperations)
		})
	}
}
