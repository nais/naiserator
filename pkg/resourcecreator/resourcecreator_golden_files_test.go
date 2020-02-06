package resourcecreator_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nsf/jsondiff"
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
	Error           *string
	Input           json.RawMessage
	ResourceOptions resourcecreator.ResourceOptions
	Output          []json.RawMessage
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

	opts := jsondiff.DefaultConsoleOptions()

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
			for i := range resources {
				nm := resources[i].Resource.GetObjectKind().GroupVersionKind().GroupKind().String()
				t.Run(nm, func(t *testing.T) {

					resourceJSON, err := json.Marshal(resources[i])
					if err != nil {
						t.Errorf("unable to marshal resource %d: %s", i, err)
						t.Fail()
					}

					result, diff := jsondiff.Compare(resourceJSON, test.Output[i], &opts)

					switch {
					case result == jsondiff.FullMatch:
						return
					case result == jsondiff.SupersetMatch && test.Config.MatchType == "subset":
						return
					default:
						t.Error(diff)
					}
				})
			}
		}
	})
}

func TestResourceCreator(t *testing.T) {
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
		path := filepath.Join(testDataDirectory, name)
		t.Run(name, func(t *testing.T) {
			subTest(t, path)
		})
	}
}
