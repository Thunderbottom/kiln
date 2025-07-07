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
			name:  "quoted values",
			input: `QUOTED_KEY="quoted value"` + "\n",
			expected: map[string][]byte{
				"QUOTED_KEY": []byte("quoted value"),
			},
		},
		{
			name:  "values with spaces",
			input: "SPACE_KEY=value with spaces\n",
			expected: map[string][]byte{
				"SPACE_KEY": []byte("value with spaces"),
			},
		},
		{
			name:        "invalid format",
			input:       "INVALID_LINE_NO_EQUALS\n",
			expectError: true,
		},
		{
			name:     "empty input",
			input:    "",
			expected: map[string][]byte{},
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

			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: expected %d, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Missing key %s", key)

					continue
				}

				if !reflect.DeepEqual(expectedValue, actualValue) {
					t.Errorf("Value mismatch for %s: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestFormatEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]byte
		expected string
	}{
		{
			name: "simple variables",
			input: map[string][]byte{
				"KEY1": []byte("value1"),
				"KEY2": []byte("value2"),
			},
		},
		{
			name: "empty value",
			input: map[string][]byte{
				"EMPTY": []byte(""),
			},
		},
		{
			name:     "empty map",
			input:    map[string][]byte{},
			expected: "",
		},
		{
			name: "special characters",
			input: map[string][]byte{
				"SPECIAL": []byte("value with spaces and symbols!@#"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatEnv(tt.input)

			if tt.expected != "" {
				if string(result) != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, string(result))
				}

				return
			}

			if len(tt.input) == 0 {
				if len(result) != 0 {
					t.Errorf("Expected empty result for empty input, got %q", string(result))
				}

				return
			}

			// For non-empty inputs, verify round-trip consistency
			parsed, err := ParseEnv(result)
			if err != nil {
				t.Fatalf("ParseEnv failed on formatted output: %v", err)
			}

			if len(parsed) != len(tt.input) {
				t.Errorf("Round trip length mismatch: expected %d, got %d", len(tt.input), len(parsed))
			}

			for key, expectedValue := range tt.input {
				actualValue, exists := parsed[key]
				if !exists {
					t.Errorf("Round trip missing key %s", key)

					continue
				}

				if !reflect.DeepEqual(expectedValue, actualValue) {
					t.Errorf("Round trip value mismatch for %s: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}
