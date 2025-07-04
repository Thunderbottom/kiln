package env

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	// envVarRegex validates environment variable names
	envVarRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

	// sensitivePatterns for masking sensitive values
	sensitivePatterns = []string{
		"password", "passwd", "pwd",
		"secret", "key", "token",
		"api", "auth", "credential",
		"private", "secure", "salt",
		"certificate", "cert", "tls",
	}
)

// ParseResult contains the result of parsing an environment file
type ParseResult struct {
	Variables map[string]string
	Errors    []ParseError
	Warnings  []string
}

// ParseError represents a parsing error with line information
type ParseError struct {
	Line    int
	Content string
	Error   string
}

func (pe ParseError) String() string {
	return fmt.Sprintf("line %d: %s (content: %q)", pe.Line, pe.Error, pe.Content)
}

// ParseEnvFile parses environment file content and returns variables with validation
func ParseEnvFile(content string) (map[string]string, error) {
	result := ParseEnvFileDetailed(content)

	// Return errors if any critical parsing errors occurred
	if len(result.Errors) > 0 {
		return result.Variables, fmt.Errorf("parsing errors: %v", result.Errors)
	}

	return result.Variables, nil
}

// ParseEnvFileDetailed parses with detailed error reporting
func ParseEnvFileDetailed(content string) ParseResult {
	result := ParseResult{
		Variables: make(map[string]string),
		Errors:    []ParseError{},
		Warnings:  []string{},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		originalLine := line
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line
		if err := parseLine(line, originalLine, lineNum, &result); err != nil {
			result.Errors = append(result.Errors, ParseError{
				Line:    lineNum,
				Content: originalLine,
				Error:   err.Error(),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		result.Errors = append(result.Errors, ParseError{
			Line:    lineNum,
			Content: "",
			Error:   fmt.Sprintf("scanner error: %v", err),
		})
	}

	return result
}

// parseLine parses a single line of environment variables
func parseLine(line, originalLine string, lineNum int, result *ParseResult) error {
	// Find the first = sign
	eqIndex := strings.Index(line, "=")
	if eqIndex == -1 {
		return fmt.Errorf("missing '=' separator")
	}

	key := strings.TrimSpace(line[:eqIndex])
	value := line[eqIndex+1:] // Don't trim value to preserve leading/trailing spaces

	// Validate key
	if err := ValidateEnvVarName(key); err != nil {
		return fmt.Errorf("invalid variable name: %v", err)
	}

	// Check for duplicate keys
	if _, exists := result.Variables[key]; exists {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("line %d: duplicate variable %q (previous value will be overwritten)", lineNum, key))
	}

	// Process value (handle quotes, escapes, etc.)
	processedValue, err := processValue(value)
	if err != nil {
		return fmt.Errorf("invalid value: %v", err)
	}

	result.Variables[key] = processedValue
	return nil
}

// processValue handles quote removal and escape sequences
func processValue(value string) (string, error) {
	value = strings.TrimSpace(value)

	if len(value) == 0 {
		return "", nil
	}

	// Handle quoted values
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]

		// Double quotes
		if first == '"' && last == '"' {
			return processDoubleQuotedValue(value[1 : len(value)-1])
		}

		// Single quotes (no escape processing)
		if first == '\'' && last == '\'' {
			return value[1 : len(value)-1], nil
		}
	}

	// Unquoted value - check for problematic characters
	if strings.ContainsAny(value, "\n\r\t") {
		return "", fmt.Errorf("unquoted value contains whitespace characters")
	}

	return value, nil
}

// processDoubleQuotedValue handles escape sequences in double-quoted strings
func processDoubleQuotedValue(value string) (string, error) {
	var result strings.Builder
	result.Grow(len(value))

	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			switch value[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 't':
				result.WriteByte('\t')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			default:
				// Unknown escape sequence - keep as is
				result.WriteByte('\\')
				result.WriteByte(value[i+1])
			}
			i++ // Skip the escaped character
		} else {
			result.WriteByte(value[i])
		}
	}

	return result.String(), nil
}

// ValidateEnvVarName validates an environment variable name
func ValidateEnvVarName(name string) error {
	if name == "" {
		return fmt.Errorf("empty variable name")
	}

	if !envVarRegex.MatchString(name) {
		return fmt.Errorf("invalid characters (must match [A-Za-z_][A-Za-z0-9_]*)")
	}

	// Check for reserved names
	if isReservedName(name) {
		return fmt.Errorf("reserved variable name")
	}

	return nil
}

// isReservedName checks if a variable name is reserved
func isReservedName(name string) bool {
	reserved := []string{
		"PATH", "HOME", "USER", "SHELL", "PWD", "OLDPWD",
		"TMPDIR", "TMP", "TEMP", "EDITOR", "PAGER",
		"LD_LIBRARY_PATH", "LD_PRELOAD",
	}

	for _, r := range reserved {
		if strings.EqualFold(name, r) {
			return true
		}
	}

	return false
}

// IsSensitiveKey checks if an environment variable key appears to contain sensitive data
func IsSensitiveKey(key string) bool {
	key = strings.ToLower(key)

	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}

	return false
}

// MaskSensitiveValue masks sensitive values for display
func MaskSensitiveValue(key, value string) string {
	if !IsSensitiveKey(key) {
		return value
	}

	if len(value) == 0 {
		return ""
	}

	if len(value) <= 8 {
		return "****"
	}

	// Show first 2 and last 2 characters
	return value[:2] + "****" + value[len(value)-2:]
}

// FormatEnvFile formats environment variables back to file format
func FormatEnvFile(vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}

	var lines []string

	// Add header comment
	lines = append(lines, "# Environment Variables")
	lines = append(lines, "# Generated by kiln")
	lines = append(lines, "")

	// Sort keys for consistent output
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}

	// Simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		value := vars[key]
		lines = append(lines, formatEnvLine(key, value))
	}

	return strings.Join(lines, "\n") + "\n"
}

// formatEnvLine formats a single environment variable line
func formatEnvLine(key, value string) string {
	// Quote value if it contains special characters
	if needsQuoting(value) {
		value = `"` + escapeValue(value) + `"`
	}

	return fmt.Sprintf("%s=%s", key, value)
}

// needsQuoting checks if a value needs to be quoted
func needsQuoting(value string) bool {
	if value == "" {
		return false
	}

	// Quote if contains whitespace or special characters
	for _, r := range value {
		if unicode.IsSpace(r) || r == '"' || r == '\'' || r == '\\' || r == '$' {
			return true
		}
	}

	return false
}

// escapeValue escapes special characters in a value
func escapeValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return value
}
