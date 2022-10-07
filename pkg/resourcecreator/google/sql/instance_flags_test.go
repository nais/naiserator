package google_sql

import (
	"testing"
)

func TestBoolTrue(t *testing.T) {
	flagName := "auto_explain.log_analyze"
	flagValue := "true"
	err := ValidateFlag(flagName, flagValue)
	if err != nil {
		t.Error(err)
	}
}

func TestBoolBogus(t *testing.T) {
	flagName := "auto_explain.log_analyze"
	flagValue := "bogus"
	err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Errorf("'%s' is not within spec", flagValue)
	}
}

func TestStringWithinEnum(t *testing.T) {
	flagName := "auto_explain.log_format"
	flagValue := "json"
	err := ValidateFlag(flagName, flagValue)
	if err != nil {
		t.Error(err)
	}
}

func TestStringNotWithinEnum(t *testing.T) {
	flagName := "auto_explain.log_format"
	flagValue := "bogus"
	err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Errorf("'%s' is not within spec", flagValue)
	}
}

func TestIntWithinRange(t *testing.T) {
	flagName := "commit_siblings"
	flagValue := "2"
	err := ValidateFlag(flagName, flagValue)
	if err != nil {
		t.Error(err)
	}
}

func TestIntNotWithinRange(t *testing.T) {
	flagName := "commit_siblings"
	flagValue := "1001"
	err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Errorf("'%s' is not within spec", flagValue)
	}
}

func TestFloatWithinRange(t *testing.T) {
	flagName := "autovacuum_vacuum_scale_factor"
	flagValue := "10"
	err := ValidateFlag(flagName, flagValue)
	if err != nil {
		t.Error(err)
	}
}

func TestFloatNotWithinRange(t *testing.T) {
	flagName := "autovacuum_vacuum_scale_factor"
	flagValue := "-1"
	err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Errorf("'%s' is not within spec", flagValue)
	}
}

func TestIsUnitOf(t *testing.T) {
	flagName := "effective_cache_size"
	flagValue := "24576"
	err := ValidateFlag(flagName, flagValue)
	if err != nil {
		t.Error(err)
	}
}

func TestIsNotUnitOf(t *testing.T) {
	flagName := "effective_cache_size"
	flagValue := "24277"
	err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Errorf("'%s' is not within spec", flagValue)
	}
}

func TestIsNotEmpty(t *testing.T) {
	flagName := "pglogical.conflict_log_level"
	flagValue := ""
	err := ValidateFlag(flagName, flagValue)
	if err == nil {
		t.Errorf("'%s' is not within spec", flagValue)
	}
}