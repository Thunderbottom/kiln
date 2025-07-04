package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/thunderbottom/kiln/internal/config"
	"gopkg.in/yaml.v3"
)

func Export(ctx context.Context, args []string) error {
	// Check for help flags first
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printExportUsage()
			return nil
		}
	}

	fs := flag.NewFlagSet("export", flag.ContinueOnError)

	file := fs.String("file", "default", "environment file to export")
	format := fs.String("format", "shell", "output format: shell, json, yaml, env, table")
	mask := fs.Bool("mask", false, "mask sensitive values")
	verbose := fs.Bool("v", false, "verbose output")

	fs.Usage = printExportUsage

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	// Handle positional argument for file
	if fs.NArg() > 0 {
		*file = fs.Arg(0)
	}

	// Validate format
	validFormats := map[string]bool{
		"shell": true, "json": true, "yaml": true, "env": true, "table": true,
	}
	if !validFormats[*format] {
		return fmt.Errorf("unsupported format: %s (supported: shell, json, yaml, env, table)", *format)
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get environment file path
	envFilePath := cfg.GetEnvFile(*file)
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("environment file not found: %s", envFilePath)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Exporting from: %s (%s)\n", *file, envFilePath)
	}

	// Load and decrypt environment variables
	envVars, err := loadAndDecryptEnv(ctx, cfg, envFilePath, *verbose)
	if err != nil {
		return err
	}

	if len(envVars) == 0 {
		if *verbose {
			fmt.Fprintf(os.Stderr, "No variables to export\n")
		}
		return nil
	}

	// Process environment variables (common logic)
	processedVars := processEnvVars(envVars, cfg, *mask)

	// Output in requested format using strategy pattern
	formatter := getFormatter(*format)
	return formatter(processedVars)
}

// printExportUsage prints the usage information for the export command
func printExportUsage() {
	fmt.Print(`Usage: kiln export [flags] [file]

Export decrypted environment variables in various formats.

Flags:
  -file string     environment file to export (default "default")
  -format string   output format: shell, json, yaml, env, table (default "shell")
  -mask            mask sensitive values
  -v               verbose output

Formats:
  shell   Shell export commands (default)
  json    JSON object
  yaml    YAML format
  env     Plain KEY=value format
  table   Human-readable table

Examples:
  kiln export
  kiln export --format json
  kiln export staging --format yaml
  kiln export --mask
  eval $(kiln export)  # Source into current shell
`)
}

// processEnvVars handles the common processing logic for all formats
func processEnvVars(envVars map[string]string, cfg *config.Config, mask bool) map[string]string {
	processed := make(map[string]string)

	for key, value := range envVars {
		if mask && cfg.IsSensitiveKey(key) {
			value = maskSensitiveValue(key, value)
		}
		processed[key] = value
	}

	return processed
}

// FormatterFunc represents a function that formats environment variables
type FormatterFunc func(map[string]string) error

// getFormatter returns the appropriate formatter function for the given format
func getFormatter(format string) FormatterFunc {
	switch format {
	case "shell":
		return formatShell
	case "json":
		return formatJSON
	case "yaml":
		return formatYAML
	case "env":
		return formatEnv
	case "table":
		return formatTable
	default:
		return func(map[string]string) error {
			return fmt.Errorf("unsupported format: %s", format)
		}
	}
}

// formatShell outputs shell export commands
func formatShell(envVars map[string]string) error {
	keys := getSortedKeys(envVars)
	for _, key := range keys {
		value := envVars[key]
		escapedValue := shellEscape(value)
		fmt.Printf("export %s=%s\n", key, escapedValue)
	}
	return nil
}

// formatJSON outputs JSON format
func formatJSON(envVars map[string]string) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(envVars)
}

// formatYAML outputs YAML format
func formatYAML(envVars map[string]string) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(envVars)
}

// formatEnv outputs plain KEY=value format
func formatEnv(envVars map[string]string) error {
	keys := getSortedKeys(envVars)
	for _, key := range keys {
		fmt.Printf("%s=%s\n", key, envVars[key])
	}
	return nil
}

// formatTable outputs human-readable table format
func formatTable(envVars map[string]string) error {
	if len(envVars) == 0 {
		fmt.Println("No environment variables found.")
		return nil
	}

	keys := getSortedKeys(envVars)

	// Calculate column widths
	maxKeyLen := len("VARIABLE")
	maxValueLen := len("VALUE")

	for _, key := range keys {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
		value := envVars[key]
		if len(value) > maxValueLen {
			maxValueLen = len(value)
		}
	}

	// Limit value column width for readability
	if maxValueLen > 50 {
		maxValueLen = 50
	}

	// Print header
	fmt.Printf("┌─%s─┬─%s─┐\n",
		strings.Repeat("─", maxKeyLen),
		strings.Repeat("─", maxValueLen))
	fmt.Printf("│ %-*s │ %-*s │\n", maxKeyLen, "VARIABLE", maxValueLen, "VALUE")
	fmt.Printf("├─%s─┼─%s─┤\n",
		strings.Repeat("─", maxKeyLen),
		strings.Repeat("─", maxValueLen))

	// Print rows
	for _, key := range keys {
		value := envVars[key]
		// Truncate long values
		if len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}
		fmt.Printf("│ %-*s │ %-*s │\n", maxKeyLen, key, maxValueLen, value)
	}

	// Print footer
	fmt.Printf("└─%s─┴─%s─┘\n",
		strings.Repeat("─", maxKeyLen),
		strings.Repeat("─", maxValueLen))

	fmt.Printf("\nTotal: %d variables\n", len(envVars))
	return nil
}

func getSortedKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func shellEscape(value string) string {
	// For shell safety, always quote values
	if strings.Contains(value, "'") {
		// If value contains single quotes, use double quotes and escape
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\"", "\\\"")
		value = strings.ReplaceAll(value, "$", "\\$")
		value = strings.ReplaceAll(value, "`", "\\`")
		return `"` + value + `"`
	}

	// Use single quotes for simplicity
	return `'` + value + `'`
}
