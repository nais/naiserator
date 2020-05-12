package deepcomp_test

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/nais/naiserator/pkg/test/deepcomp"
	"github.com/stretchr/testify/assert"
)

type testcase struct {
	expected string
	actual   string
	mode     deepcomp.MatchType
	diffset  deepcomp.Diffset
}

var testcases = []testcase{
	// string comparison
	{
		expected: `"foo"`,
		actual:   `"foo"`,
		mode:     `exact`,
	},
	{
		expected: `"foo"`,
		actual:   `"bar"`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".",
				Type:    "valueDiffers",
				Message: "expected string 'foo' but got string 'bar'",
			},
		},
	},
	{
		expected: `"foo"`,
		actual:   `123`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".",
				Type:    "typeDiffers",
				Message: "expected string 'foo' but got float64 '123'",
			},
		},
	},
	// struct comparison
	{
		expected: `{"foo":"bar"}`,
		actual:   `{"foo":"bar"}`,
		mode:     `exact`,
	},
}

func decode(data string) interface{} {
	i := new(interface{})
	err := json.Unmarshal([]byte(data), &i)
	if err != nil {
		panic(fmt.Errorf("error in test fixture: %s", err))
	}
	return i
}

func TestSubset(t *testing.T) {
	var test testcase
	var i int

	defer func() {
		err := recover()
		if err != nil {
			t.Errorf("panic in test %d: %s", i+1, err)
			debug.PrintStack()
			t.Fail()
		}
	}()

	for i, test = range testcases {
		a := decode(test.expected)
		b := decode(test.actual)
		if test.diffset == nil {
			test.diffset = deepcomp.Diffset{}
		}
		diffs := deepcomp.Compare(test.mode, a, b)
		assert.Equal(t, test.diffset, diffs, "test %d", i+1)
	}
}
