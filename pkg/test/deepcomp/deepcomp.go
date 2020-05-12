// deep comparison provides normalized diffs between arbitrary data structures.
// test framework to ease reading and writing golden test fixtures.

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

func Compare(matchType MatchType, expected, actual interface{}) Diffset {
	switch matchType {
	case MatchRegex:
		panic("")
	case MatchExact:
		return Exact(expected, actual, matchType)
	case MatchSubset:
		return Subset(expected, actual)
	default:
		panic(fmt.Errorf("unhandled type %v", matchType))
	}
}

func Subset(expected, actual interface{}) Diffset {

	diffs := Exact(expected, actual, MatchSubset)
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
func Exact(x, y interface{}, matchType MatchType) Diffset {
	if x == nil || y == nil {
		return Diffset{}
	}
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	return deepValueEqual(v1, v2, 0, "", matchType)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func subsliceequal(a, b reflect.Value, depth int, path string, matchType MatchType) Diffset {
	diffs := make(Diffset, 0)
	for i := 0; i < a.Len(); i++ {
		if i >= b.Len() {
			return append(diffs, Diff{
				Path:    path,
				Message: fmt.Sprintf("expected %d but got %d", a.Len(), b.Len()),
				Type:    DiffMissingField,
			})
		}
		diffs = append(diffs, deepValueEqual(a.Index(i), b.Index(i), depth+1, fmt.Sprintf("%s[%d]", path, i), matchType)...)
	}
	if a.Len() < b.Len() {
		return append(diffs, Diff{
			Path:    path,
			Message: fmt.Sprintf("expected %d but got %d", a.Len(), b.Len()),
			Type:    DiffExtraField,
		})
	}
	return diffs
}

func subslice(a, b reflect.Value, depth int, path string, matchType MatchType) Diffset {
	switch matchType {
	case MatchSubset:
		return subslicesubset(a, b, depth, path, matchType)
	default:
		return subsliceequal(a, b, depth, path, matchType)
	}
}
func subslicesubset(a, b reflect.Value, depth int, path string, matchType MatchType) Diffset {
	diffs := make(Diffset, 0)
	alen, blen := a.Len(), b.Len()
	max := max(alen, blen)
	ai, bi := 0, 0

	if alen == 0 {
		return Diffset{}
	}

	for bi = 0; bi < max; bi++ {
		if ai >= alen || bi >= blen {
			break
		}
		diffs = deepValueEqual(a.Index(ai), b.Index(bi), depth+1, fmt.Sprintf("%s[%d]", path, ai), matchType)
		if len(diffs) == 0 { // increment expected array only if matches actual
			ai++
		}
	}

	if ai == alen && bi == blen {
		return diffs
	}

	if ai < alen {
		elem := a.Index(ai).Elem()
		return Diffset{Diff{
			Path:    path,
			Message: fmt.Sprintf("expected %s '%+v' but reached end of input without finding it", elem.Kind().String(), elem.Interface()),
			Type:    DiffMissingField,
		}}
	}

	return Diffset{}
}

// Tests for deep equality using reflected types. The map argument tracks
// comparisons that have already been seen, which allows short circuiting on
// recursive types.
func deepValueEqual(a, b reflect.Value, depth int, path string, matchType MatchType) Diffset {
	diffs := make(Diffset, 0)
	simpleExpect := Diff{
		Path:    path,
		Message: fmt.Sprintf("expected %s '%+v' but got %s '%+v'", a.Kind().String(), a.Interface(), b.Kind().String(), b.Interface()),
		Type:    DiffValueDiffers,
	}

	if !a.IsValid() || !b.IsValid() {
		simpleExpect.Type = DiffInvalidTypes
		return Diffset{simpleExpect}
	}

	if a.Type() != b.Type() {
		simpleExpect.Type = DiffTypeDiffers
		return Diffset{simpleExpect}
	}

	switch a.Kind() {
	case reflect.Array:
		for i := 0; i < a.Len(); i++ {
			diffs = append(diffs, deepValueEqual(a.Index(i), b.Index(i), depth+1, fmt.Sprintf("%s[%d]", path, i), matchType)...)
		}
	case reflect.Slice:
		if a.IsNil() != b.IsNil() {
			return append(diffs, simpleExpect)
		}
		if a.Pointer() == b.Pointer() {
			return diffs
		}
		diffs = append(diffs, subslice(a, b, depth, path, matchType)...)
	case reflect.Interface:
		if a.IsNil() != b.IsNil() {
			return append(diffs, simpleExpect)
		}
		return deepValueEqual(a.Elem(), b.Elem(), depth+1, path, matchType)
	case reflect.Ptr:
		if a.Pointer() != b.Pointer() {
			return deepValueEqual(a.Elem(), b.Elem(), depth+1, path, matchType)
		}
		/*
			case reflect.Struct:
				for i, n := 0, v1.NumField(); i < n; i++ {
					ds = append(ds, deepValueEqual(v1.Field(i), v2.Field(i), depth+1, ds)...)
				}
		*/
	case reflect.Map:
		if a.IsNil() != b.IsNil() {
			return append(diffs, simpleExpect)
		}
		if a.Pointer() == b.Pointer() {
			return diffs
		}
		v1keys := a.MapKeys()
		for _, k := range v1keys {
			val1 := a.MapIndex(k)
			val2 := b.MapIndex(k)
			if !val2.IsValid() {
				diffs = append(diffs, Diff{
					Path:    path + "." + k.String(),
					Message: "missing map value",
					Type:    DiffMissingField,
				})
			} else {
				diffs = append(diffs, deepValueEqual(val1, val2, depth+1, path+"."+k.String(), matchType)...)
			}
		}
		if a.Len() == b.Len() {
			return diffs
		}
		// too many values
		for _, k := range b.MapKeys() {
			if !a.MapIndex(k).IsValid() {
				diffs = append(diffs, Diff{
					Path:    path + "." + k.String(),
					Message: "unexpected map key",
					Type:    DiffExtraField,
				})
			}
		}
	default:
		// Normal equality suffices
		// FIXME: support regular expressions
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			diffs = append(diffs, simpleExpect)
		}
	}
	return diffs
}
