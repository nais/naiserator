// deep comparison provides diffs between arbitrary data structures.
package deepcomp

import (
	"fmt"
	"reflect"
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
		return Exact(expected, actual)
	case MatchSubset:
		return Subset(expected, actual)
	default:
		panic(fmt.Errorf("unhandled type %v", typ))
	}
}

func Subset(expected, actual interface{}) Diffset {

	diffs := Exact(expected, actual)
	subset := make(Diffset, 0, len(diffs))

	for _, diff := range diffs {
		if diff.Type != DiffExtraField {
			subset = append(subset, diff)
		}
	}

	return subset
}

// As Exact traverses the data values it may find a cycle. The
// second and subsequent times that Exact compares two pointer
// values that have been compared before, it treats the values as
// equal rather than examining the values to which they point.
// This ensures that Exact terminates.
func Exact(x, y interface{}) Diffset {
	if x == nil || y == nil {
		return Diffset{}
	}
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	return deepValueEqual(v1, v2, 0, "")
}

// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueEqual(v1, v2 reflect.Value, depth int, path string) Diffset {
	ds := make(Diffset, 0)
	simpleExpect := Diff{
		Path:    path,
		Message: fmt.Sprintf("expected %s '%+v' but got %s '%+v'", v1.Kind().String(), v1.Interface(), v2.Kind().String(), v2.Interface()),
		Type:    DiffValueDiffers,
	}

	if !v1.IsValid() || !v2.IsValid() {
		simpleExpect.Type = DiffInvalidTypes
		return Diffset{simpleExpect}
	}

	if v1.Type() != v2.Type() {
		simpleExpect.Type = DiffTypeDiffers
		return Diffset{simpleExpect}
	}

	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			ds = append(ds, deepValueEqual(v1.Index(i), v2.Index(i), depth+1, fmt.Sprintf("%s[%d]", path, i))...)
		}
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			return append(ds, simpleExpect)
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
			ds = append(ds, deepValueEqual(v1.Index(i), v2.Index(i), depth+1, fmt.Sprintf("%s[%d]", path, i))...)
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
			return append(ds, simpleExpect)
		}
		return deepValueEqual(v1.Elem(), v2.Elem(), depth+1, path)
	case reflect.Ptr:
		if v1.Pointer() != v2.Pointer() {
			return deepValueEqual(v1.Elem(), v2.Elem(), depth+1, path)
		}
		/*
			case reflect.Struct:
				for i, n := 0, v1.NumField(); i < n; i++ {
					ds = append(ds, deepValueEqual(v1.Field(i), v2.Field(i), depth+1, ds)...)
				}
		*/
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return append(ds, simpleExpect)
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
				ds = append(ds, deepValueEqual(val1, val2, depth+1, path+"."+k.String())...)
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
			ds = append(ds, simpleExpect)
		}
	}
	return ds
}
