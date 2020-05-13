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
	name     string
	expected string
	actual   string
	mode     deepcomp.MatchType
	diffset  deepcomp.Diffset
}

var testcases = []testcase{
	{
		name:     `string comparison`,
		expected: `"foo"`,
		actual:   `"foo"`,
		mode:     `exact`,
	},
	{
		name:     `string comparison fail`,
		expected: `"foo"`,
		actual:   `"bar"`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "ErrValueDiffers",
				Message: "expected string 'foo' but got string 'bar'",
			},
		},
	},
	{
		name:     `string compared against number`,
		expected: `"123"`,
		actual:   `123`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "ErrTypeDiffers",
				Message: "expected string '123' but got float64 '123'",
			},
		},
	},
	{
		name:     `struct comparison`,
		expected: `{"foo":"bar"}`,
		actual:   `{"foo":"bar"}`,
		mode:     `exact`,
	},
	{
		name:     `missing values in structs`,
		expected: `{"foo":"bar"}`,
		actual:   `{"foo":"bar","bar":"baz"}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".bar",
				Type:    "ErrExtraField",
				Message: "unexpected map key",
			},
		},
	},
	{
		name:     `extra values in structs`,
		expected: `{"foo":"bar","bar":"baz"}`,
		actual:   `{"foo":"bar"}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".bar",
				Type:    "ErrMissingField",
				Message: "missing map value",
			},
		},
	},
	{
		name:     `nested structs`,
		expected: `{"foo":{"bar":{"baz":"ok"}}}`,
		actual:   `{"foo":{"bar":{"baz":"ok"}}}`,
		mode:     `exact`,
	},
	{
		name:     `nested structs comparison failed with path`,
		expected: `{"foo":{"bar":{"baz":"ok"}}}`,
		actual:   `{"foo":{"bar":{"baz":"nope"}}}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".foo.bar.baz",
				Type:    "ErrValueDiffers",
				Message: "expected string 'ok' but got string 'nope'",
			},
		},
	},
	{
		name:     `slices`,
		expected: `[1,2,3]`,
		actual:   `[1,2,3]`,
		mode:     `exact`,
	},
	{
		name:     `slices with too many elements`,
		expected: `[1,2,3]`,
		actual:   `[1,2,3,4]`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "ErrExtraField",
				Message: "expected 3 but got 4",
			},
		},
	},
	{
		name:     `slices with too few elements`,
		expected: `[1,2,3,4]`,
		actual:   `[1,2,3]`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "ErrMissingField",
				Message: "expected 4 but got 3",
			},
		},
	},
	{
		name:     `nested complex types`,
		expected: `{"foo":[0,{"bar":["baz"]},2,3]}`,
		actual:   `{"foo":[0,{"bar":["baz"]},2,3]}`,
		mode:     `exact`,
	},
	{
		name:     `nested complex types`,
		expected: `{"foo":[0,{"bro":["baz"]},2,3]}`,
		actual:   `{"foo":[0,{"bro":["foo"]},2,3]}`,
		mode:     `exact`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".foo[1].bro[0]",
				Type:    "ErrValueDiffers",
				Message: "expected string 'baz' but got string 'foo'",
			},
		},
	},
	{
		name:     `subset matching of simple structs`,
		expected: `{"foo":"bar"}`,
		actual:   `{"foo":"bar","bar":"baz"}`,
		mode:     `subset`,
	},
	{
		name:     `multiple subset matches`,
		expected: `{"foo":"bar","baz":[1,2,3]}`,
		actual:   `{"foo":"bar","bar":"baz","more":"test","baz":[1,2,3]}`,
		mode:     `subset`,
	},

	{
		name:     `subset failure`,
		expected: `{"foo":"bar","baz":[1,2]}`,
		actual:   `{"foo":"bar","bar":"baz","more":"test","baz":[1,3]}`,
		mode:     `subset`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".baz",
				Type:    "ErrMissingField",
				Message: "expected float64 '2' but reached end of input without finding it",
			},
		},
	},
	{
		name:     `slice subsets`,
		expected: `[1,3,7,9]`,
		actual:   `[1,2,3,4,5,6,7,8,9]`,
		mode:     `subset`,
	},
	{
		name:     `empty actual slices in subsets`,
		expected: `[1,2,3,4,5,6,7,8,9]`,
		actual:   `[]`,
		mode:     `subset`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "ErrMissingField",
				Message: "expected float64 '1' but reached end of input without finding it",
			},
		},
	},
	{
		name:     `empty slice subsets`,
		expected: `[]`,
		actual:   `[1,2,3,4,5,6,7,8,9]`,
		mode:     `subset`,
	},
	{
		name:     `too many slice subset expectations`,
		expected: `[1,2,3,4,5,6,7,8,9]`,
		actual:   `[1,2,3,4]`,
		mode:     `subset`,
		diffset: deepcomp.Diffset{
			{
				Path:    "",
				Type:    "ErrMissingField",
				Message: "expected float64 '5' but reached end of input without finding it",
			},
		},
	},
	{
		name:     `complex slice subsets`,
		expected: `[{"foo":"bar"},{"bar":"baz"}]`,
		actual:   `[{"pre":"pre"},{"foo":"bar"},{"bar":"baz"},{"post":"post"}]`,
		mode:     `subset`,
	},
	{
		name:     `regular expression matching`,
		expected: `{"foo":"[abc][123]"}`,
		actual:   `{"foo":"c1"}`,
		mode:     `regex`,
	},
	{
		name:     `regular expression matching failure`,
		expected: `{"foo":"[abc]56"}`,
		actual:   `{"foo":"c1"}`,
		mode:     `regex`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".foo",
				Type:    "ErrValueDiffers",
				Message: "regular expression \"[abc]56\" doesn't match value \"c1\"",
			},
		},
	},
	{
		name:     `regular expression syntax error`,
		expected: `{"foo":"[ab"}`,
		actual:   `{"foo":"c1"}`,
		mode:     `regex`,
		diffset: deepcomp.Diffset{
			{
				Path:    ".foo",
				Type:    "ErrInvalidRegex",
				Message: "error parsing regexp: missing closing ]: `[ab\"`",
			},
		},
	},
	{
		name:     `regular expression matching as subset`,
		expected: `[{"foo":"[abc][123]"}]`,
		actual:   `[{"foo":"c1"},{"bar":"bar"},1,2,3]`,
		mode:     `regex`,
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
		assert.Equal(t, test.diffset, diffs, "test: %s", test.name)
	}
}
