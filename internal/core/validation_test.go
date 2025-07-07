package core

import "testing"

func TestIsValidVarName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid uppercase", "DATABASE_URL", true},
		{"valid lowercase", "database_url", true},
		{"valid mixed case", "Database_Url", true},
		{"valid with numbers", "API_KEY_123", true},
		{"valid starting underscore", "_PRIVATE_VAR", true},
		{"invalid dash", "API-KEY", false},
		{"empty string", "", false},
		{"starts with number", "123_KEY", false},
		{"starts with special char", "#VAR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidVarName(tt.input); got != tt.want {
				t.Errorf("IsValidVarName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidEnvValue(t *testing.T) {
	tests := []struct {
		name    string
		value   []byte
		wantErr bool
	}{
		{"normal value", []byte("normal value"), false},
		{"empty value", []byte(""), false},
		{"with newline", []byte("line1\nline2"), false},
		{"null byte", []byte("bad\x00value"), true},
		{"too large", make([]byte, 2*1024*1024), true}, // 2MB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidEnvValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsValidEnvValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
