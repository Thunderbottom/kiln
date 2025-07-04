package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/crypto"
	"github.com/thunderbottom/kiln/internal/env"
	"github.com/thunderbottom/kiln/internal/errors"
	"github.com/thunderbottom/kiln/internal/utils"
	"gopkg.in/yaml.v3"
)

var (
	exportFile    string
	exportFormat  string
	exportMask    bool
	exportFilter  []string
	exportExclude []string
)

var exportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export environment variables",
	Long:  GetLongDescription("export", ExportDescription),
	RunE:  runExport,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []string
		for name := range cfg.ListEnvFiles() {
			completions = append(completions, name)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportFile, "file", "",
		"environment file to export (default: from config)")
	exportCmd.Flags().StringVar(&exportFormat, "format", "shell",
		"output format: shell, json, yaml, env, table")
	exportCmd.Flags().BoolVar(&exportMask, "mask", false,
		"mask sensitive values")
	exportCmd.Flags().StringSliceVar(&exportFilter, "filter", nil,
		"only export variables matching these prefixes")
	exportCmd.Flags().StringSliceVar(&exportExclude, "exclude", nil,
		"exclude variables with these names")

	// Register flag completions
	exportCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		formats := []string{"shell", "json", "yaml", "env", "table"}
		return formats, cobra.ShellCompDirectiveNoFileComp
	})
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return errors.Wrap(err, "failed to load configuration")
	}

	// Determine which file to export
	envFile := exportFile
	if envFile == "" {
		if len(args) > 0 {
			envFile = args[0]
		} else {
			envFile = "default"
		}
	}

	envFilePath := cfg.GetEnvFile(envFile)

	// Check if file exists
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("environment file not found: %s", envFilePath)
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Exporting from: %s (%s)\n", envFile, envFilePath)
	}

	// Load private key
	privateKey, keyInfo, err := utils.LoadPrivateKey(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to load private key")
	}

	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "Using private key from: %s\n", keyInfo.Source)
	}

	// Setup age manager
	ageManager, err := crypto.NewAgeManager(cfg.Recipients)
	if err != nil {
		return errors.Wrap(err, "failed to setup encryption")
	}

	if err := ageManager.AddIdentity(privateKey); err != nil {
		return errors.Wrap(err, "failed to add identity")
	}

	// Read and decrypt file
	encrypted, err := os.ReadFile(envFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read environment file")
	}

	plaintext, err := ageManager.Decrypt(encrypted)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt environment file")
	}

	// Parse environment variables
	envVars, err := env.ParseEnvFile(string(plaintext))
	if err != nil {
		return errors.Wrap(err, "failed to parse environment file")
	}

	// Apply filters
	filteredVars := applyFilters(envVars, exportFilter, exportExclude)

	if len(filteredVars) == 0 {
		if IsVerbose() {
			fmt.Fprintf(os.Stderr, "No variables to export after filtering\n")
		}
		return nil
	}

	// Output in requested format
	switch exportFormat {
	case "shell":
		outputShellFormat(filteredVars, exportMask)
	case "json":
		if err := outputJSONFormat(filteredVars, exportMask); err != nil {
			return err
		}
	case "yaml":
		if err := outputYAMLFormat(filteredVars, exportMask); err != nil {
			return err
		}
	case "env":
		outputEnvFormat(filteredVars, exportMask)
	case "table":
		outputTableFormat(filteredVars, exportMask)
	default:
		return fmt.Errorf("unsupported format: %s (supported: shell, json, yaml, env, table)", exportFormat)
	}

	return nil
}

func applyFilters(envVars map[string]string, filters, excludes []string) map[string]string {
	result := make(map[string]string)

	for key, value := range envVars {
		// Check excludes first
		excluded := false
		for _, exclude := range excludes {
			if key == exclude || strings.HasPrefix(key, exclude) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check filters
		if len(filters) > 0 {
			included := false
			for _, filter := range filters {
				if key == filter || strings.HasPrefix(key, filter) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		result[key] = value
	}

	return result
}

func outputShellFormat(envVars map[string]string, mask bool) {
	keys := getSortedKeys(envVars)

	for _, key := range keys {
		value := envVars[key]
		if mask {
			value = env.MaskSensitiveValue(key, value)
		}

		// Escape value for shell
		escapedValue := shellEscape(value)
		fmt.Printf("export %s=%s\n", key, escapedValue)
	}
}

func outputJSONFormat(envVars map[string]string, mask bool) error {
	output := make(map[string]string)

	for key, value := range envVars {
		if mask {
			value = env.MaskSensitiveValue(key, value)
		}
		output[key] = value
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputYAMLFormat(envVars map[string]string, mask bool) error {
	output := make(map[string]string)

	for key, value := range envVars {
		if mask {
			value = env.MaskSensitiveValue(key, value)
		}
		output[key] = value
	}

	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(output)
}

func outputEnvFormat(envVars map[string]string, mask bool) {
	keys := getSortedKeys(envVars)

	for _, key := range keys {
		value := envVars[key]
		if mask {
			value = env.MaskSensitiveValue(key, value)
		}

		// Simple format without shell escaping
		fmt.Printf("%s=%s\n", key, value)
	}
}

func outputTableFormat(envVars map[string]string, mask bool) {
	if len(envVars) == 0 {
		fmt.Println("No environment variables found.")
		return
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
		if mask {
			value = env.MaskSensitiveValue(key, value)
		}

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
		if mask {
			value = env.MaskSensitiveValue(key, value)
		}

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
