package deepcomp

import (
	"fmt"
	"reflect"
)

type MatchType string

type ErrorType string

const (
	MatchRegex  MatchType = "regex"
	MatchExact  MatchType = "exact"
	MatchSubset MatchType = "subset"

	ErrMissingField ErrorType = "ErrMissingField"
	ErrExtraField   ErrorType = "ErrExtraField"
	ErrTypeDiffers  ErrorType = "ErrTypeDiffers"
	ErrValueDiffers ErrorType = "ErrValueDiffers"
	ErrInvalidTypes ErrorType = "ErrInvalidTypes"
	ErrInvalidRegex ErrorType = "ErrInvalidRegex"
)

type Diffset []Diff

type Diff struct {
	Path    string
	Message string
	Type    ErrorType
	A       *reflect.Value
	B       *reflect.Value
}

func (diff Diff) String() string {
	return fmt.Sprintf("%s at %s: %s", diff.Type, diff.Path, diff.Message)
}

func (diffs Diffset) Filter(errorType ErrorType) Diffset {
	matched := make(Diffset, 0, len(diffs))
	for _, diff := range diffs {
		if diff.Type != errorType {
			matched = append(matched, diff)
		}
	}
	return matched
}
