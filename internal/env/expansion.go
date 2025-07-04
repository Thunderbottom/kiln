package env

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// ExpandVariables performs simple variable expansion in environment values
func ExpandVariables(envVars map[string]string, allowCommands bool) map[string]string {
	expanded := make(map[string]string)

	// Set up environment for expansion
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	// Expand each variable
	for key, value := range envVars {
		result := value

		// Handle command substitution first if allowed
		if allowCommands {
			result = expandCommands(result)
		}

		// Then handle variable expansion using Go's built-in function
		result = os.ExpandEnv(result)

		expanded[key] = result
	}

	return expanded
}

// expandCommands handles $(command) substitution
func expandCommands(value string) string {
	// Simple regex for $(command) that doesn't interfere with ${var}
	cmdRegex := regexp.MustCompile(`\$\(([^)]+)\)`)

	return cmdRegex.ReplaceAllStringFunc(value, func(match string) string {
		command := match[2 : len(match)-1] // Remove $( and )

		cmd := exec.Command("/bin/sh", "-c", command)
		output, err := cmd.Output()
		if err != nil {
			return "" // Return empty on error
		}

		return strings.TrimSuffix(string(output), "\n")
	})
}
