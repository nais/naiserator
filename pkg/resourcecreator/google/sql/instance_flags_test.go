package google_sql

import (
	"testing"
)

func TestValidateFlag(t *testing.T) {
	tests := []struct {
		name      string
		flagName  string
		flagValue string
		wantError bool
	}{
		{
			name:      "bool true",
			flagName:  "auto_explain.log_analyze",
			flagValue: "on",
		},
		{
			name:      "bool false",
			flagName:  "auto_explain.log_analyze",
			flagValue: "off",
		},
		{
			name:      "bool bogus",
			flagName:  "auto_explain.log_analyze",
			flagValue: "bogus",
			wantError: true,
		},
		{
			name:      "single value within enum",
			flagName:  "auto_explain.log_format",
			flagValue: "json",
		},
		{
			name:      "multiple values all within enum",
			flagName:  "auto_explain.log_format",
			flagValue: "json,xml",
		},
		{
			name:      "multiple values only some within enum",
			flagName:  "auto_explain.log_format",
			flagValue: "json,xml, bogus",
			wantError: true,
		},
		{
			name:      "string not within enum",
			flagName:  "auto_explain.log_format",
			flagValue: "bogus",
			wantError: true,
		},
		{
			name:      "int within range",
			flagName:  "commit_siblings",
			flagValue: "2",
		},
		{
			name:      "int not within range",
			flagName:  "commit_siblings",
			flagValue: "1001",
			wantError: true,
		},
		{
			name:      "float within range",
			flagName:  "autovacuum_vacuum_scale_factor",
			flagValue: "10",
		},
		{
			name:      "float not within range",
			flagName:  "autovacuum_vacuum_scale_factor",
			flagValue: "-1",
			wantError: true,
		},
		{
			name:      "is unit of",
			flagName:  "effective_cache_size",
			flagValue: "24576",
		},
		{
			name:      "is not unit of",
			flagName:  "effective_cache_size",
			flagValue: "24277",
			wantError: true,
		},
		{
			name:      "is not empty",
			flagName:  "pglogical.conflict_log_level",
			flagValue: "",
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFlag(tt.flagName, tt.flagValue)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
