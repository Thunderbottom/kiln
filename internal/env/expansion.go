package env

import (
	"maps"
	"os"
	"regexp"
	"strings"
)

// ExpandVariables performs simple variable expansion in environment values
func ExpandVariables(envVars map[string]string) map[string]string {
	expanded := make(map[string]string)

	// Create combined environment for lookups
	lookup := make(map[string]string)

	// Add system environment
	for _, env := range os.Environ() {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			lookup[parts[0]] = parts[1]
		}
	}

	// Add kiln environment (takes precedence)
	maps.Copy(lookup, envVars)

	// Expand each variable
	for key, value := range envVars {
		expanded[key] = expandValue(value, lookup)
	}

	return expanded
}

// expandValue performs simple environment variable substitution
func expandValue(value string, lookup map[string]string) string {
	// Simple ${VAR} expansion only
	varRegex := regexp.MustCompile(`\$\{([^}]+)\}`)

	return varRegex.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if envValue, exists := lookup[varName]; exists {
			return envValue
		}
		return "" // Return empty string for undefined variables
	})
}
