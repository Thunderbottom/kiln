package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// SetCmd represents the set command for adding or updating environment variables.
type SetCmd struct {
	Name     string `arg:"" help:"Environment variable name" optional:""`
	Value    string `arg:"" help:"Environment variable value (if not provided, will prompt for input)" optional:""`
	File     string `short:"f" help:"Environment file to modify" default:"default"`
	FromFile string `help:"JSON file containing environment variables to set" type:"path"`
}

func (c *SetCmd) validate() error {
	if c.FromFile != "" && c.Name != "" {
		return kerrors.ValidationError("arguments", "cannot use both --from-file and variable name argument")
	}

	if c.FromFile == "" && c.Name == "" {
		return kerrors.ValidationError("arguments", "must provide either variable name or --from-file")
	}

	if c.Name != "" && !core.IsValidVarName(c.Name) {
		return kerrors.ValidationError("variable name", "name is required")
	}

	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	if c.FromFile != "" {
		if !core.IsValidFilePath(c.FromFile) {
			return kerrors.ValidationError("JSON file path", "invalid file path")
		}

		if !core.FileExists(c.FromFile) {
			return kerrors.ValidationError("JSON file", "file does not exist")
		}
	}

	return nil
}

// Run executes the set command, prompting for and storing environment variable(s).
func (c *SetCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "set").Str("file", c.File).Bool("from_file", c.FromFile != "").Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	identity, err := rt.Identity()
	if err != nil {
		return err
	}

	cfg, err := rt.Config()
	if err != nil {
		return err
	}

	if c.FromFile != "" {
		return c.setFromFile(rt, identity, cfg)
	}

	return c.setSingleVariable(rt, identity, cfg)
}

// setFromFile handles setting multiple variables from JSON file
func (c *SetCmd) setFromFile(rt *Runtime, identity *core.Identity, cfg *config.Config) error {
	rt.Logger.Debug().Str("json_file", c.FromFile).Msg("parsing JSON file")

	variables, parseErr := c.parseJSONFile()
	if parseErr != nil {
		return parseErr
	}

	if err := c.validateJSONVariables(variables); err != nil {
		return err
	}

	rt.Logger.Debug().Int("variable_count", len(variables)).Msg("parsed variables from JSON")

	existingVars, cleanup, err := core.GetAllEnvVars(identity, cfg, c.File)
	if err != nil {
		return err
	}
	defer cleanup()

	mergedVars := make(map[string][]byte)

	for key, value := range existingVars {
		newValue := make([]byte, len(value))
		copy(newValue, value)
		mergedVars[key] = newValue
	}

	overwriteCount := 0

	for key, value := range variables {
		if _, exists := mergedVars[key]; exists {
			overwriteCount++
		}

		mergedVars[key] = value
	}

	if err := core.SaveAllEnvVars(identity, cfg, c.File, mergedVars); err != nil {
		return err
	}

	rt.Logger.Info().Str("file", c.File).Str("source", c.FromFile).
		Int("added", len(variables)-overwriteCount).
		Int("updated", overwriteCount).
		Int("total", len(mergedVars)).
		Msg("variables set from JSON file")

	return nil
}

// setSingleVariable handles setting a single variable
func (c *SetCmd) setSingleVariable(rt *Runtime, identity *core.Identity, cfg *config.Config) error {
	var value []byte
	if c.Value != "" {
		value = []byte(c.Value)
	} else {
		var err error

		value, err = c.readValueFromStdin()
		if err != nil {
			return kerrors.InputError("stdin", "failed to read value", "ensure terminal supports password input")
		}
	}
	defer core.WipeData(value)

	if err := core.IsValidEnvValue(value); err != nil {
		return kerrors.ValidationError("variable value", err.Error())
	}

	value = core.SanitizeEnvValue(value)

	if err := core.SetEnvVar(identity, cfg, c.File, c.Name, value); err != nil {
		return err
	}

	rt.Logger.Info().Str("file", c.File).Str("variable", c.Name).Msg("set successfully")

	return nil
}

// parseJSONFile reads and parses JSON file containing environment variables
func (c *SetCmd) parseJSONFile() (map[string][]byte, error) {
	data, err := os.ReadFile(c.FromFile)
	if err != nil {
		return nil, kerrors.FileAccessError("read", c.FromFile, err)
	}

	var jsonVars map[string]any
	if err := json.Unmarshal(data, &jsonVars); err != nil {
		return nil, kerrors.ValidationError("JSON format", fmt.Sprintf("invalid JSON in file '%s': %s", c.FromFile, err.Error()))
	}

	variables := make(map[string][]byte)

	for key, value := range jsonVars {
		if !core.IsValidVarName(key) {
			return nil, kerrors.ValidationError("variable name",
				fmt.Sprintf("'%s' must start with letter or underscore, followed by letters, numbers, or underscores", key))
		}

		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case bool:
			strValue = fmt.Sprintf("%t", v)
		case float64:
			if v == float64(int64(v)) {
				strValue = fmt.Sprintf("%.0f", v)
			} else {
				strValue = fmt.Sprintf("%g", v)
			}
		case nil:
			strValue = ""
		default:
			return nil, kerrors.ValidationError("variable value",
				fmt.Sprintf("unsupported value type for '%s': %T", key, value))
		}

		valueBytes := []byte(strValue)

		if err := core.IsValidEnvValue(valueBytes); err != nil {
			return nil, kerrors.ValidationError("variable value",
				fmt.Sprintf("invalid value for '%s': %s", key, err.Error()))
		}

		variables[key] = core.SanitizeEnvValue(valueBytes)
	}

	return variables, nil
}

// validateJSONVariables performs additional validation on the parsed variables
func (c *SetCmd) validateJSONVariables(variables map[string][]byte) error {
	if len(variables) == 0 {
		return kerrors.ValidationError("JSON content", "no valid environment variables found")
	}

	if len(variables) > 1000 {
		return kerrors.ValidationError("JSON content", "too many variables (max 1000)")
	}

	return nil
}

// readValueFromStdin prompts for and reads a value from stdin with hidden input
func (c *SetCmd) readValueFromStdin() ([]byte, error) {
	fmt.Fprintf(os.Stderr, "Enter value for %s: ", c.Name)

	// Convert to int since syscall.Stdin is not int on Windows
	//nolint:unconvert
	value, err := term.ReadPassword(int(syscall.Stdin))

	fmt.Println()

	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	return value, nil
}
