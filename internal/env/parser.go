package env

import (
	"bufio"
	"fmt"
	"strings"
)

// ParseEnvFile parses environment file content and returns variables
func ParseEnvFile(content string) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: invalid format, expected KEY=value", lineNum)
		}

		key := strings.TrimSpace(parts[0])
		value := parts[1] // Don't trim value to preserve leading/trailing spaces

		// Basic key validation
		if key == "" {
			return nil, fmt.Errorf("line %d: empty variable name", lineNum)
		}

		// Handle quoted values (simple approach)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
				// Handle basic escape sequences in double quotes
				if strings.Contains(value, "\\") {
					value = strings.ReplaceAll(value, "\\n", "\n")
					value = strings.ReplaceAll(value, "\\t", "\t")
					value = strings.ReplaceAll(value, "\\\"", "\"")
					value = strings.ReplaceAll(value, "\\'", "'")
					value = strings.ReplaceAll(value, "\\\\", "\\")
				}
			}
		}

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading content: %v", err)
	}

	return result, nil
}

// FormatEnvFile formats environment variables back to file format
func FormatEnvFile(vars map[string]string) string {
	if len(vars) == 0 {
		return "# Environment Variables\n\n"
	}

	var lines []string
	lines = append(lines, "# Environment Variables")
	lines = append(lines, "")

	// Sort keys for consistent output
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}

	// Simple sort implementation
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		value := vars[key]
		// Quote values that need it
		if needsQuoting(value) {
			value = fmt.Sprintf(`"%s"`, escapeValue(value))
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(lines, "\n") + "\n"
}

// needsQuoting checks if a value needs to be quoted
func needsQuoting(value string) bool {
	if value == "" {
		return false
	}

	// Quote if contains whitespace, quotes, or special characters
	return strings.ContainsAny(value, " \t\n\r\"'\\$`")
}

// escapeValue escapes special characters in a value
func escapeValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return value
}
