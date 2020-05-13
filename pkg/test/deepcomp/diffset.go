package deepcomp

import (
	"fmt"
)

type MatchType string

type DiffType string

const (
	MatchRegex  MatchType = "regex"
	MatchExact  MatchType = "exact"
	MatchSubset MatchType = "subset"

	ErrMissingField DiffType = "ErrMissingField"
	ErrExtraField   DiffType = "ErrExtraField"
	ErrTypeDiffers  DiffType = "ErrTypeDiffers"
	ErrValueDiffers DiffType = "ErrValueDiffers"
	ErrInvalidTypes DiffType = "ErrInvalidTypes"
	ErrInvalidRegex DiffType = "ErrInvalidRegex"
)

type Diffset []Diff

type Diff struct {
	Path    string
	Message string
	Type    DiffType
}

func (diff Diff) String() string {
	return fmt.Sprintf("%s at %s: %s", diff.Type, diff.Path, diff.Message)
}
