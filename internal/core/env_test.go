package core

import (
	"reflect"
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    map[string][]byte
		expectError bool
	}{
		{
			name:  "simple key-value pairs",
			input: "KEY1=value1\nKEY2=value2\n",
			expected: map[string][]byte{
				"KEY1": []byte("value1"),
				"KEY2": []byte("value2"),
			},
		},
		{
			name:  "empty values",
			input: "EMPTY_KEY=\n",
			expected: map[string][]byte{
				"EMPTY_KEY": []byte(""),
			},
		},
		{
			name:        "invalid format",
			input:       "INVALID_LINE_NO_EQUALS\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseEnv([]byte(tt.input))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatEnv(t *testing.T) {
	input := map[string][]byte{
		"KEY1": []byte("value1"),
		"KEY2": []byte("value2"),
	}

	result := FormatEnv(input)
	if result == nil {
		t.Fatal("FormatEnv returned nil")
	}

	// Parse back to verify
	parsed, err := ParseEnv(result)
	if err != nil {
		t.Fatalf("ParseEnv failed: %v", err)
	}

	if !reflect.DeepEqual(input, parsed) {
		t.Errorf("Round trip failed: expected %v, got %v", input, parsed)
	}
}
