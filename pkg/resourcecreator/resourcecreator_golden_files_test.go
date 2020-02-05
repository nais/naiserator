package resourcecreator_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/stretchr/testify/assert"
)

const (
	testDataDirectory = "testdata"
)

type testCaseConfig struct {
	Description string
}

type testCase struct {
	Config          testCaseConfig
	Error           *string
	Input           json.RawMessage
	ResourceOptions resourcecreator.ResourceOptions
	Output          json.RawMessage
}

// minimal kubernetes object
type minimalObject struct {
	ApiVersion string
	Kind       string
	Metadata   struct {
		Name string
	}
}

type resourceOperation struct {
	Operation string
	Resource  minimalObject
}

func shallowcompare(t *testing.T, expected, actual json.RawMessage) bool {
	var e, a []resourceOperation
	_ = json.Unmarshal(expected, &e)
	_ = json.Unmarshal(actual, &a)
	return assert.Equal(t, e, a)
}

func jsoncompare(t *testing.T, expected, actual json.RawMessage) bool {
	return assert.Equal(t, string(expected), string(actual))
}

func compare(t *testing.T, expected, actual json.RawMessage) {
	if !shallowcompare(t, expected, actual) {
		return
	}
	//jsoncompare(t, expected, actual)
}

func compactJSON(src json.RawMessage) (json.RawMessage, error) {
	var decoded interface{}
	err := json.Unmarshal(src, &decoded)
	if err != nil {
		return nil, fmt.Errorf("decode: %s", err)
	}
	return json.Marshal(decoded)
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

	out, err := compactJSON(test.Output)
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
			resourceJSON, err := json.Marshal(resources)
			assert.NoError(t, err)
			compare(t, out, resourceJSON)
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
