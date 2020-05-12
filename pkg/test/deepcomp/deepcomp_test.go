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
	// string comparison fail
	{
		expected: `"foo"`,
		actual:   `"bar"`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "valueDiffers",
				Message: "expected string 'foo' but got string 'bar'",
			},
		},
	},
	// string compared against number
	{
		expected: `"foo"`,
		actual:   `123`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
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
	// missing values in structs
	{
		expected: `{"foo":"bar"}`,
		actual:   `{"foo":"bar","bar":"baz"}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".bar",
				Type:    "extraField",
				Message: "unexpected map key",
			},
		},
	},
	// extra values in structs
	{
		expected: `{"foo":"bar","bar":"baz"}`,
		actual:   `{"foo":"bar"}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".bar",
				Type:    "missingField",
				Message: "missing map value",
			},
		},
	},
	// nested structs
	{
		expected: `{"foo":{"bar":{"baz":"ok"}}}`,
		actual:   `{"foo":{"bar":{"baz":"ok"}}}`,
		mode:     `exact`,
	},
	// nested structs comparison failed with path
	{
		expected: `{"foo":{"bar":{"baz":"ok"}}}`,
		actual:   `{"foo":{"bar":{"baz":"nope"}}}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".foo.bar.baz",
				Type:    "valueDiffers",
				Message: "expected string 'ok' but got string 'nope'",
			},
		},
	},
	// slices
	{
		expected: `[1,2,3]`,
		actual:   `[1,2,3]`,
		mode:     `exact`,
	},
	// slices with too many elements
	{
		expected: `[1,2,3]`,
		actual:   `[1,2,3,4]`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "extraField",
				Message: "too many elements; expected 3 but got 4",
			},
		},
	},
	// slices with too few elements
	{
		expected: `[1,2,3,4]`,
		actual:   `[1,2,3]`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "missingField",
				Message: "too few elements; expected 4 but got 3",
			},
		},
	},
	// nested complex types
	{
		expected: `{"foo":[0,{"bar":["baz"]},2,3]}`,
		actual:   `{"foo":[0,{"bar":["baz"]},2,3]}`,
		mode:     `exact`,
	},
	// nested complex types
	{
		expected: `{"foo":[0,{"bro":["baz"]},2,3]}`,
		actual:   `{"foo":[0,{"bro":["foo"]},2,3]}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".foo[1].bro[0]",
				Type:    "valueDiffers",
				Message: "expected string 'baz' but got string 'foo'",
			},
		},
	},
	// subset matching of simple structs
	{
		expected: `{"foo":"bar"}`,
		actual:   `{"foo":"bar","bar":"baz"}`,
		mode:     `subset`,
	},
	// multiple subset matches
	{
		expected: `{"foo":"bar","baz":[1,2,3]}`,
		actual:   `{"foo":"bar","bar":"baz","more":"test","baz":[1,2,3]}`,
		mode:     `subset`,
	},
	// subset failure
	{
		expected: `{"foo":"bar","baz":[1,2]}`,
		actual:   `{"foo":"bar","bar":"baz","more":"test","baz":[1,3]}`,
		mode:     `subset`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".baz[1]",
				Type:    "valueDiffers",
				Message: "expected float64 '2' but got float64 '3'",
			},
		},
	},
	// slice subsets
	/*
		{
			expected: `[1,3,5]`,
			actual:   `[1,2,3,4,5,6,7,8,9]`,
			mode:     `subset`,
		},
	*/
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
