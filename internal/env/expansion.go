package env

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// ExpandVariables expands ${VAR} and $(command) syntax in environment values
func ExpandVariables(envVars map[string]string, allowCommands bool) (map[string]string, error) {
	expanded := make(map[string]string)

	// Create a combined environment for lookups
	combinedEnv := make(map[string]string)

	// Add system environment
	for _, env := range os.Environ() {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			combinedEnv[parts[0]] = parts[1]
		}
	}

	// Add kiln environment (takes precedence)
	maps.Copy(combinedEnv, envVars)

	// Expand each variable
	for key, value := range envVars {
		expandedValue, err := expandValue(value, combinedEnv, allowCommands)
		if err != nil {
			return nil, fmt.Errorf("failed to expand %s: %w", key, err)
		}
		expanded[key] = expandedValue
	}

	return expanded, nil
}

func expandValue(value string, env map[string]string, allowCommands bool) (string, error) {
	// Expand ${VAR} syntax
	varRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	value = varRegex.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if envValue, exists := env[varName]; exists {
			return envValue
		}
		return "" // Return empty string for undefined variables
	})

	// Expand $(command) syntax if allowed
	if allowCommands {
		cmdRegex := regexp.MustCompile(`\$\(([^)]+)\)`)
		var err error
		value = cmdRegex.ReplaceAllStringFunc(value, func(match string) string {
			cmdStr := match[2 : len(match)-1] // Remove $( and )
			output, cmdErr := executeCommand(cmdStr)
			if cmdErr != nil {
				err = cmdErr
				return match // Return original if command fails
			}
			return strings.TrimSpace(output)
		})

		if err != nil {
			return "", err
		}
	}

	return value, nil
}

func executeCommand(cmdStr string) (string, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}
	return string(output), nil
}

// ValidateVariableReferences checks for circular references
func ValidateVariableReferences(envVars map[string]string) error {
	// Build dependency graph
	deps := make(map[string][]string)
	varRegex := regexp.MustCompile(`\$\{([^}]+)\}`)

	for key, value := range envVars {
		matches := varRegex.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			referencedVar := match[1]
			deps[key] = append(deps[key], referencedVar)
		}
	}

	// Check for circular dependencies
	for key := range envVars {
		if hasCycle(key, deps, make(map[string]bool), make(map[string]bool)) {
			return fmt.Errorf("circular dependency detected involving variable: %s", key)
		}
	}

	return nil
}

func hasCycle(node string, deps map[string][]string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range deps[node] {
		if !visited[neighbor] && hasCycle(neighbor, deps, visited, recStack) {
			return true
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}
