// deep comparison provides diffs between arbitrary data structures.
package deepcomp

import (
	"fmt"
	"reflect"

	"github.com/go-test/deep"
)

type MatchType string

type DiffType string

const (
	MatchRegex  MatchType = "regex"
	MatchExact  MatchType = "exact"
	MatchSubset MatchType = "subset"

	DiffMissingField DiffType = "missingField"
	DiffExtraField   DiffType = "extraField"
	DiffTypeDiffers  DiffType = "typeDiffers"
	DiffValueDiffers DiffType = "valueDiffers"
	DiffInvalidTypes DiffType = "invalidTypes"
)

type Diffset []Diff

type Diff struct {
	Path    string
	Message string
	Type    DiffType
}

func Compare(typ MatchType, expected, actual interface{}) Diffset {
	switch typ {
	case MatchRegex:
		panic("")
	case MatchExact:
		return DeepEqual(expected, actual)
		return Exact(expected, actual, "", Diffset{})
	case MatchSubset:
		return Subset(expected, actual, "")
	default:
		panic(fmt.Errorf("unhandled type %v", typ))
	}
}

// As DeepEqual traverses the data values it may find a cycle. The
// second and subsequent times that DeepEqual compares two pointer
// values that have been compared before, it treats the values as
// equal rather than examining the values to which they point.
// This ensures that DeepEqual terminates.
func DeepEqual(x, y interface{}) Diffset {
	ds := Diffset{}
	if x == nil || y == nil {
		return ds
	}
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return append(ds, Diff{
			Path:    ".",
			Message: fmt.Sprintf("type differs: expected %s but got %s", v1.Type().String(), v1.Type().String()),
			Type:    "",
		})
	}
	return deepValueEqual(v1, v2, 0, "", ds)
}

// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueEqual(v1, v2 reflect.Value, depth int, path string, ds Diffset) Diffset {
	brp := ".foo[1].bro[0]"
	if brp == path {
		path = brp
	}
	expected := Diff{
		Path:    path,
		Message: fmt.Sprintf("expected %s '%+v' but got %s '%+v'", v1.Kind().String(), v1.Interface(), v2.Kind().String(), v2.Interface()),
		Type:    DiffValueDiffers,
	}

	if !v1.IsValid() || !v2.IsValid() {
		expected.Type = DiffInvalidTypes
		return append(ds, expected)
	}

	if v1.Type() != v2.Type() {
		expected.Type = DiffTypeDiffers
		return append(ds, expected)
	}

	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			ds = deepValueEqual(v1.Index(i), v2.Index(i), depth+1, fmt.Sprintf("%s[%d]", path, i), ds)
		}
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			return append(ds, expected)
		}
		if v1.Pointer() == v2.Pointer() {
			return ds
		}
		for i := 0; i < v1.Len(); i++ {
			if i >= v2.Len() {
				ds = append(ds, Diff{
					Path:    path,
					Message: fmt.Sprintf("too few elements; expected %d but got %d", v1.Len(), v2.Len()),
					Type:    DiffMissingField,
				})
				break
			}
			ds = deepValueEqual(v1.Index(i), v2.Index(i), depth+1, fmt.Sprintf("%s[%d]", path, i), ds)
		}
		if v1.Len() < v2.Len() {
			ds = append(ds, Diff{
				Path:    path,
				Message: fmt.Sprintf("too many elements; expected %d but got %d", v1.Len(), v2.Len()),
				Type:    DiffExtraField,
			})
		}
	case reflect.Interface:
		if v1.IsNil() != v2.IsNil() {
			return append(ds, expected)
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), depth+1, path, ds)
	case reflect.Ptr:
		if v1.Pointer() != v2.Pointer() {
			return deepValueEqual(v1.Elem(), v2.Elem(), depth+1, path, ds)
		}
		/*
			case reflect.Struct:
				for i, n := 0, v1.NumField(); i < n; i++ {
					ds = append(ds, deepValueEqual(v1.Field(i), v2.Field(i), depth+1, ds)...)
				}
		*/
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return append(ds, expected)
		}
		if v1.Pointer() == v2.Pointer() {
			return ds
		}
		for _, k := range v1.MapKeys() {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val2.IsValid() {
				ds = append(ds, Diff{
					Path:    path + "." + k.String(),
					Message: "missing map value",
					Type:    DiffMissingField,
				})
			} else {
				ds = deepValueEqual(val1, val2, depth+1, path+"."+k.String(), ds)
			}
		}
		if v1.Len() == v2.Len() {
			return ds
		}
		// too many values
		for _, k := range v2.MapKeys() {
			if !v1.MapIndex(k).IsValid() {
				ds = append(ds, Diff{
					Path:    path + "." + k.String(),
					Message: "unexpected map key",
					Type:    DiffExtraField,
				})
			}
		}
	default:
		// Normal equality suffices
		if !reflect.DeepEqual(v1.Interface(), v2.Interface()) {
			ds = append(ds, expected)
		}
	}
	return ds
}

func Exact(expected, actual interface{}, path string, ds Diffset) Diffset {

	ta, tb := reflect.TypeOf(expected), reflect.TypeOf(actual)
	ka, kb := ta.Kind(), tb.Kind()
	_ = reflect.ValueOf(expected).Convert(reflect.TypeOf(expected)).Interface()
	// tb := reflect.ValueOf(actual).Convert(reflect.TypeOf(actual)).Interface()

	if ka != kb {
		// return fmt.Errorf("%s: value is of type %T, expected %T", path, kb.String(), ka.String())
	}

	switch ka {
	case reflect.String:
		err := stringcmp(expected.(string), actual.(string))
		if err != nil {
			ds = append(ds, Diff{
				Path:    path,
				Message: err.Error(),
			})
		}
	// case map[string]interface{}:
	// err = subsetTest(t, expected, actual, path + ".")
	case reflect.Map:
		// ma, mb := ta.(map[string]interface{}), tb.(map[string]interface{})
		ra := reflect.ValueOf(expected).MapRange()
		for ra.Next() {
			// k, v := ra.Key(), ra.Value()
			// rb := reflect.ValueOf(actual).MapIndex(k)
			// err = Subset(v.Interface(), rb.Interface(), path+"."+k.String())
		}
	default:
		ds = append(ds, Diff{
			Path:    path,
			Message: "reached default case",
		})
	}

	return ds
}

func deepCompare(expected, actual reflect.Value) error {
	// FIXME: mandag: ikke bruk reflect
	var err error

	if !expected.IsValid() || !actual.IsValid() {
		if expected.IsValid() == actual.IsValid() {
			return nil
		}
		return fmt.Errorf("validity differs")
	}

	kind := expected.Kind()

	switch kind {
	case reflect.Map:
		for _, k := range expected.MapKeys() {
			val1 := expected.MapIndex(k)
			val2 := actual.MapIndex(k)
			err = deepCompare(val1, val2)
			if err != nil {
				return fmt.Errorf("sub: %s", err)
			}
		}

	case reflect.Interface:
		if expected.IsNil() || actual.IsNil() {
			if expected.IsNil() == actual.IsNil() {
				return nil
			}
			return fmt.Errorf("%s: interfaces differ in nil values", expected.String())
		}
		return deepCompare(expected.Elem(), actual.Elem())
	}

	return nil
}

func stringcmp(expected, actual string) error {
	if expected == actual {
		return nil
	}
	return fmt.Errorf("strings differ")
}

func Subset(expected, actual interface{}, path string) Diffset {

	diffs := deep.Equal(expected, actual)
	if len(diffs) == 0 {
		return nil
	}
	return Diffset{
		Diff{
			Path:    "",
			Message: diffs[0],
		},
	}

}
