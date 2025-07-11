package commands

import (
	"fmt"
	"os"
	"regexp"

	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

type ApplyCmd struct {
	File           string `short:"f" help:"Environment file from configuration" required:"" placeholder="KILN-ENV-FILE" default:"default"`
	Output         string `short:"o" help:"Output file path (default: stdout)"`
	Strict         bool   `help:"Fail if template variables are not found"`
	LeftDelimiter  string `help:"Left delimiter to use for template variables (default: ${ or $)"`
	RightDelimiter string `help:"Right delimiter to use for template variables (default: } or empty)"`
	Template       string `arg:"" help:"Template file path" required:""`
}

func (c *ApplyCmd) validate() error {
	if !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	if !core.IsValidFilePath(c.Template) {
		return kerrors.ValidationError("template path", "invalid file path")
	}

	if c.Output != "" && !core.IsValidFilePath(c.Output) {
		return kerrors.ValidationError("output path", "invalid file path")
	}

	if (c.LeftDelimiter != "" && c.RightDelimiter == "") || (c.LeftDelimiter == "" && c.RightDelimiter != "") {
		return kerrors.ValidationError("delimiters", "both left and right delimiters must be specified together")
	}

	return nil
}

// Run executes the apply command, substituting variables in the template file.
func (c *ApplyCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "apply").Str("file", c.File).Str("template", c.Template).Msg("validation started")

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

	variables, cleanup, err := core.GetAllEnvVars(identity, cfg, c.File)
	if err != nil {
		return err
	}
	defer cleanup()

	templateContent, err := os.ReadFile(c.Template)
	if err != nil {
		return kerrors.FileAccessError("read", c.Template, err)
	}

	result, err := c.substituteVariables(templateContent, variables)
	if err != nil {
		return err
	}

	if c.Output != "" {
		return os.WriteFile(c.Output, result, 0644)
	}

	fmt.Print(string(result))
	return nil
}

// buildPatterns creates regex patterns based on delimiter configuration.
func (c *ApplyCmd) buildPatterns() ([]*regexp.Regexp, error) {
	var patterns []*regexp.Regexp

	if c.LeftDelimiter != "" && c.RightDelimiter != "" {
		leftEscaped := regexp.QuoteMeta(c.LeftDelimiter)
		rightEscaped := regexp.QuoteMeta(c.RightDelimiter)
		customPattern := regexp.MustCompile(leftEscaped + `\s*([A-Za-z_][A-Za-z0-9_]*)\s*` + rightEscaped)
		patterns = append(patterns, customPattern)
	} else {
		bracesPattern := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
		simplePattern := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
		patterns = append(patterns, bracesPattern, simplePattern)
	}

	return patterns, nil
}

// substituteVariables performs variable substitution in template content.
func (c *ApplyCmd) substituteVariables(content []byte, variables map[string][]byte) ([]byte, error) {
	patterns, err := c.buildPatterns()
	if err != nil {
		return nil, err
	}

	var missingVars []string
	result := content

	for _, pattern := range patterns {
		result = pattern.ReplaceAllFunc(result, func(match []byte) []byte {
			submatches := pattern.FindSubmatch(match)
			if len(submatches) < 2 {
				return match
			}

			varName := string(submatches[1])
			if value, exists := variables[varName]; exists {
				return value
			}

			if c.Strict {
				missingVars = append(missingVars, varName)
			}
			return match
		})
	}

	if len(missingVars) > 0 {
		uniqueVars := removeDuplicates(missingVars)
		return nil, kerrors.ValidationError("missing variables", fmt.Sprintf("variables not found: %v", uniqueVars))
	}

	return result, nil
}

// removeDuplicates removes duplicate strings from a slice.
func removeDuplicates(strs []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, str := range strs {
		if !keys[str] {
			keys[str] = true
			result = append(result, str)
		}
	}

	return result
}
