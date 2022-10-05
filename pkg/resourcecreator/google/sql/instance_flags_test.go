package google_sql

import (
	"fmt"
	"testing"
)

func TestBoolTrue(t *testing.T) {
	flagName := "auto_explain.log_analyze"
	flagValue := "true"
	valid, err := ValidateFlag(flagName, flagValue)
	if err != nil || !valid {
		t.Fatalf("'%s' should be a valid value for '%s'", flagValue, flagName)
	}
}

func TestBoolBogus(t *testing.T) {
	flagName := "auto_explain.log_analyze"
	flagValue := "bogus"
	_, err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Fatalf("'%s' should not be a valid value for '%s'", flagValue, flagName)
	}
}

func TestStringWithinEnum(t *testing.T) {
	flagName := "auto_explain.log_format"
	flagValue := "json"
	valid, err := ValidateFlag(flagName, flagValue)
	if err != nil || !valid {
		t.Fatalf("'%s' should be a valid value for '%s'", flagValue, flagName)
	}
}

func TestStringNotWithinEnum(t *testing.T) {
	flagName := "auto_explain.log_format"
	flagValue := "bogus"
	valid, err := ValidateFlag(flagName, flagValue)
	fmt.Printf("%v %v\n", valid, err)
	if err != nil || valid {
		t.Fatalf("'%s' should not be a valid value for '%s'", flagValue, flagName)
	}
}

func TestIntWithinRange(t *testing.T) {
	flagName := "commit_siblings"
	flagValue := "2"
	valid, err := ValidateFlag(flagName, flagValue)
	if !valid || err != nil {
		t.Fatalf("'%s' should be a valid value for '%s'", flagValue, flagName)
	}
}

func TestIntNotWithinRange(t *testing.T) {
	flagName := "commit_siblings"
	flagValue := "1001"
	valid, _ := ValidateFlag(flagName, flagValue)
	if valid {
		t.Fatalf("'%s' should not be a valid value for '%s'", flagValue, flagName)
	}
}

func TestFloatWithinRange(t *testing.T) {
	flagName := "autovacuum_vacuum_scale_factor"
	flagValue := "10"
	valid, _ := ValidateFlag(flagName, flagValue)
	if !valid {
		t.Fatalf("'%s' should be a valid value for '%s'", flagValue, flagName)
	}
}

func TestFloatNotWithinRange(t *testing.T) {
	flagName := "autovacuum_vacuum_scale_factor"
	flagValue := "-1"
	valid, _ := ValidateFlag(flagName, flagValue)
	if valid {
		t.Fatalf("'%s' should not be a valid value for '%s'", flagValue, flagName)
	}
}

func TestIsUnitOf(t *testing.T) {
	flagName := "effective_cache_size"
	flagValue := "24576"
	valid, err := ValidateFlag(flagName, flagValue)
	if !valid || err != nil {
		t.Fatalf("'%s' should be a valid value for '%s'", flagValue, flagName)
	}
}

func TestIsNotUnitOf(t *testing.T) {
	flagName := "effective_cache_size"
	flagValue := "24277"
	valid, _ := ValidateFlag(flagName, flagValue)
	if valid {
		t.Fatalf("'%s' should not be a valid value for '%s'", flagValue, flagName)
	}
}

func TestIsNotEmpty(t *testing.T) {
	flagName := "pglogical.conflict_log_level"
	flagValue := ""
	valid, _ := ValidateFlag(flagName, flagValue)
	if valid {
		t.Fatalf("'%s' should not be a valid value for '%s'", flagValue, flagName)
	}
}
