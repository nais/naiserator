package deepcomp

import (
	"bytes"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
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
	w := new(bytes.Buffer)
	_, _ = fmt.Fprintf(w, "%s at %s: %s\n", diff.Type, diff.Path, diff.Message)
	enc := yaml.NewEncoder(w)

	_, _ = fmt.Fprintf(w, "--- expected:\n")
	_ = enc.Encode(diff.A.Interface())

	// make a new encoder to avoid document separator '---'
	enc = yaml.NewEncoder(w)
	_, _ = fmt.Fprintf(w, "+++ actual:\n")
	_ = enc.Encode(diff.B.Interface())

	return w.String()
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

func (diffs Diffset) String() string {
	w := new(bytes.Buffer)
	for _, diff := range diffs {
		w.WriteString(diff.String())
		w.WriteString("\n")
	}
	return w.String()
}
