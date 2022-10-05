package google_sql

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolTrue(t *testing.T) {
	value := "true"
	valid, err := ValidateFlag("auto_explain.log_analyze", value)
	if err != nil || !valid {
		t.Fatalf("true should be a valid bool")
	}
}

func TestBoolBogus(t *testing.T) {
	value := "bogus"
	_, err := ValidateFlag("auto_explain.log_analyze", value)
	if err == nil {
		assert.Fail(t, "non-boolean value should raise error")
	}
}

func TestStringWithinEnum(t *testing.T) {
	value := "json"
	valid, err := ValidateFlag("auto_explain.log_format", value)
	if err != nil || !valid {
		assert.Fail(t, "this instance flag should be valid")
	}
}

func TestStringNotWithinEnum(t *testing.T) {
	value := "bogus"
	valid, err := ValidateFlag("auto_explain.log_format", value)
	fmt.Printf("%v %v\n", valid, err)
	if err != nil || valid {
		assert.Fail(t, "enum does not contain 'bogus', this instance flag should not be valid")
	}
}

func TestIntWithinRange(t *testing.T) {
	value := "2"
	valid, err := ValidateFlag("commit_siblings", value)
	if !valid || err != nil {
		assert.Fail(t, "nr is within range, this instance flag should be valid")
	}
}

func TestIntNotWithinRange(t *testing.T) {
	value := "1001"
	valid, _ := ValidateFlag("commit_siblings", value)
	if valid {
		assert.Fail(t, "nr is not within range, this instance flag should not be valid")
	}
}

func TestFloatWithinRange(t *testing.T) {
	value := "10"
	valid, _ := ValidateFlag("autovacuum_vacuum_scale_factor", value)
	if !valid {
		assert.Fail(t, "nr is within range, this instance flag should be valid")
	}
}

func TestFloatNotWithinRange(t *testing.T) {
	value := "-1"
	valid, _ := ValidateFlag("autovacuum_vacuum_scale_factor", value)
	if valid {
		assert.Fail(t, "nr is not within range, this instance flag should not be valid")
	}
}

func TestIsUnitOf(t *testing.T) {
	value := "24576"
	valid, err := ValidateFlag("effective_cache_size", value)
	if !valid || err != nil {
		assert.Fail(t, "this instance flag should be valid")
	}
}

func TestIsNotUnitOf(t *testing.T) {
	value := "24277"
	valid, _ := ValidateFlag("effective_cache_size", value)
	if valid {
		assert.Fail(t, "this instance flag should not be valid")
	}
}
