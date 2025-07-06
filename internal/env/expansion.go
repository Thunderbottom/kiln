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
		_ = os.Setenv(key, value)
	}

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

// expandCommands handles $(command) substitution.
// This is dangerous and should be used with extreme caution
// by the end user!
func expandCommands(value string) string {
	cmdRegex := regexp.MustCompile(`\$\(([^)]+)\)`)

	return cmdRegex.ReplaceAllStringFunc(value, func(match string) string {
		// Remove $( and )
		command := match[2 : len(match)-1]

		cmd := exec.Command("/bin/sh", "-c", command)
		output, err := cmd.Output()
		if err != nil {
			// Return an empty output
			return ""
		}

		return strings.TrimSuffix(string(output), "\n")
	})
}
