package resourcecreator_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nsf/jsondiff"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	testDataDirectory = "testdata"
)

type testCaseConfig struct {
	Description string
	MatchType   string
}

type testCase struct {
	Config          testCaseConfig
	ResourceOptions resourcecreator.ResourceOptions
	Error           *string
	Input           json.RawMessage
	Output          []json.RawMessage
}

type match struct {
	distance int
	err      error
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

func subTest(t *testing.T, file string) {
	fixture := fileReader(file)
	data, err := ioutil.ReadAll(fixture)
	if err != nil {
		t.Errorf("unable to read test data: %s", err)
		t.Fail()
	}

	test := testCase{}
	err = json.Unmarshal(data, &test)
	if err != nil {
		t.Errorf("unable to parse unmarshal test data: %s", err)
		t.Fail()
	}

	if test.Output != nil && err != nil {
		t.Errorf("unable to re-encode test data: %s", err)
		t.Fail()
	}

	t.Run(test.Config.Description, func(t *testing.T) {
		app := &nais.Application{}
		err = json.Unmarshal(test.Input, app)
		if err != nil {
			t.Errorf("unable to unmarshal Application object: %s", err)
			t.Fail()
		}

		err = nais.ApplyDefaults(app)
		if err != nil {
			t.Errorf("apply default values to Application object: %s", err)
			t.Fail()
		}

		resources, err := resourcecreator.Create(app, test.ResourceOptions)
		if test.Error != nil {
			assert.EqualError(t, err, *test.Error)
		} else {
			assert.NoError(t, err)

			err := comparedeep(test, resources)
			if err != nil {
				t.Error(err)
				t.Fail()
			}
		}
	})
}

// given two Kubernetes objects, calculate the likelyhood that they might refer to the same object.
func distance(a, b json.RawMessage) int {
	distance := 0
	ma, mb := meta{}, meta{}
	_ = json.Unmarshal(a, &ma)
	_ = json.Unmarshal(b, &mb)
	if ma.Operation != mb.Operation {
		distance++
	}
	if ma.Resource.ApiVersion != mb.Resource.ApiVersion {
		distance++
	}
	if ma.Resource.Metadata.Name != mb.Resource.Metadata.Name {
		distance += 2
	}
	if ma.Resource.Kind != mb.Resource.Kind {
		distance += 5
	}
	return distance
}

// compare two Kubernetes objects against one another.
func compare(test testCase, a, b json.RawMessage) match {
	opts := jsondiff.DefaultConsoleOptions()
	result, diff := jsondiff.Compare(a, b, &opts)

	switch {
	case result == jsondiff.FullMatch:
		return match{}
	case result == jsondiff.SupersetMatch && test.Config.MatchType == "subset":
		return match{}
	default:
		return match{
			distance: distance(a, b),
			err:      errors.New(diff),
		}
	}
}

// Compare a real-world set of Kubernetes objects an expected set of Kubernetes objects.
// The function ignores array order and uses heuristics to figure out the most likely error message.
func comparedeep(test testCase, resources resourcecreator.ResourceOperations) error {
	var err error

	serialized := make([]json.RawMessage, len(resources))
	for i := range resources {
		serialized[i], err = json.Marshal(resources[i])
		if err != nil {
			return fmt.Errorf("unable to marshal resource %d: %s", i, err)
		}
	}

	matched := make([]bool, len(serialized))

OUTER:
	for i := range test.Output {
		errs := make([]match, 0)

		for j := range resources {
			if matched[j] {
				continue
			}
			matched[j] = true
			match := compare(test, serialized[j], test.Output[i])
			if match.err == nil {
				continue OUTER
			} else {
				errs = append(errs, match)
			}
		}

		// In case of match error against all possible array indices, return the diff
		// that is most likely the object tried matching against.
		sort.Slice(errs, func(i, j int) bool {
			return errs[i].distance < errs[j].distance
		})
		return errs[0].err
	}

	return nil
}

func TestResourceCreator(t *testing.T) {
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
		path := filepath.Join(testDataDirectory, name)
		t.Run(name, func(t *testing.T) {
			subTest(t, path)
		})
	}
}
