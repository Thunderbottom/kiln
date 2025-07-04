package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// HelpText provides centralized management of command help text
type HelpText struct {
	examples []string
	tips     []string
}

// NewHelpText creates a new help text builder
func NewHelpText() *HelpText {
	return &HelpText{
		examples: []string{},
		tips:     []string{},
	}
}

// AddExample adds an example to the help text
func (h *HelpText) AddExample(example string) *HelpText {
	h.examples = append(h.examples, example)
	return h
}

// AddTip adds a tip to the help text
func (h *HelpText) AddTip(tip string) *HelpText {
	h.tips = append(h.tips, tip)
	return h
}

// Build constructs the final help text
func (h *HelpText) Build(description string) string {
	var parts []string

	if description != "" {
		parts = append(parts, description)
	}

	if len(h.examples) > 0 {
		parts = append(parts, "")
		parts = append(parts, "Examples:")
		for _, example := range h.examples {
			parts = append(parts, "  "+example)
		}
	}

	if len(h.tips) > 0 {
		parts = append(parts, "")
		parts = append(parts, "Tips:")
		for _, tip := range h.tips {
			parts = append(parts, "  • "+tip)
		}
	}

	return strings.Join(parts, "\n")
}

// Command help text definitions
var helpTexts = map[string]*HelpText{
	"root": NewHelpText().
		AddExample("kiln init                    # Initialize new project").
		AddExample("kiln edit                    # Edit environment variables").
		AddExample("kiln export                  # Export variables for shell").
		AddExample("kiln run -- [COMMAND]        # Run command with environment").
		AddTip("Use 'kiln <command> --help' for detailed command information").
		AddTip("Set KILN_PRIVATE_KEY_FILE environment variable to specify key location"),

	"init": NewHelpText().
		AddExample("kiln init                                    # Generate new key pair").
		AddExample("kiln init --from age1xyz...                 # Use existing public key").
		AddExample("kiln init --config custom.yaml              # Custom config file").
		AddExample("kiln init --key-output ~/.config/kiln/      # Custom key location").
		AddTip("Store your private key securely after initialization").
		AddTip("Share the public key with team members for collaboration"),

	"edit": NewHelpText().
		AddExample("kiln edit                          # Edit default environment file").
		AddExample("kiln edit --file staging           # Edit staging environment").
		AddExample("kiln edit --editor nano            # Use nano editor").
		AddExample("kiln edit --no-validate            # Skip validation").
		AddTip("Set EDITOR environment variable for your preferred editor").
		AddTip("Use KILN_EDITOR to override the default editor for kiln only"),

	"export": NewHelpText().
		AddExample("kiln export                              # Export as shell commands").
		AddExample("kiln export --format json               # Export as JSON").
		AddExample("kiln export staging --format yaml       # Export staging env as YAML").
		AddExample("kiln export --mask                       # Mask sensitive values").
		AddExample("kiln export --filter API_,DB_           # Only export variables starting with API_ or DB_").
		AddExample("kiln export --exclude DEBUG,TEMP        # Exclude specific variables").
		AddExample("eval $(kiln export)                      # Source into current shell").
		AddTip("Use 'eval $(kiln export)' to load variables into your current shell").
		AddTip("The --mask flag helps when sharing output or debugging"),

	"run": NewHelpText().
		AddExample("kiln run -- env                           # Show all environment variables").
		AddExample("kiln run -- [COMMAND]                     # Run terraform with secrets").
		AddExample("kiln run --file staging -- kubectl apply  # Use staging environment").
		AddExample("kiln run --dry-run -- env                 # Show what would be injected").
		AddExample("kiln run --timeout 5m -- long-command     # Set command timeout").
		AddExample("kiln run --workdir /tmp -- pwd            # Run in specific directory").
		AddTip("Use --dry-run to preview environment variables before execution").
		AddTip("Commands inherit all current environment variables plus kiln variables"),
}

// GetLongDescription returns the long description for a command
func GetLongDescription(command, baseDescription string) string {
	if helpText, exists := helpTexts[command]; exists {
		return helpText.Build(baseDescription)
	}
	return baseDescription
}

// Common descriptions
const (
	RootDescription = `Kiln is a secure environment variable management tool that uses Age encryption
to protect sensitive configuration data. It provides encrypted storage, in-memory
decryption, and secure injection of environment variables into processes.`

	InitDescription = `Initialize a new kiln project by generating an Age key pair and creating
a configuration file. The private key will be saved securely to a file with 
restricted permissions.`

	EditDescription = `Edit encrypted environment variables in your default editor. The file will be
decrypted to a secure temporary location, opened in your editor, and re-encrypted
when you save and exit.

The editor respects the following precedence:
  1. --editor flag
  2. KILN_EDITOR environment variable  
  3. EDITOR environment variable
  4. vi (fallback)`

	ExportDescription = `Export decrypted environment variables in various formats. By default,
exports as shell commands that can be sourced into your current shell.

Supported formats:
  shell  - Shell export commands (default)
  json   - JSON object
  yaml   - YAML format  
  env    - Plain KEY=value format
  table  - Human-readable table`

	RunDescription = `Run a command with environment variables loaded from encrypted file.
The variables are decrypted in memory and injected into the process environment.`
)

var (
	cfgFile string
	verbose bool

	rootCmd = &cobra.Command{
		Use:     "kiln",
		Short:   "Secure environment variable management tool",
		Long:    GetLongDescription("root", RootDescription),
		Version: getVersion(),
	}
)

func getVersion() string {
	if version == "dev" {
		return fmt.Sprintf("dev (%s/%s, %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
	}
	return fmt.Sprintf("%s (%s)", version, commit[:8])
}

// Execute executes the root command with context support
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is .kiln.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	// Set version template
	rootCmd.SetVersionTemplate(`{{printf "%s\n" .Version}}`)

	// Enable shell completion
	rootCmd.CompletionOptions.DisableDefaultCmd = false
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(".kiln")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")

		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(home)
		}
	}

	// Read from environment
	viper.SetEnvPrefix("KILN")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err == nil && IsVerbose() {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

// IsVerbose returns true if verbose mode is enabled
func IsVerbose() bool {
	return verbose || viper.GetBool("verbose")
}

// Build variables set by -ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)
