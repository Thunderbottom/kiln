// Package commands implements all CLI commands for the kiln secure environment variable management tool.
// It provides subcommands for initializing projects, editing encrypted files, running commands with
// decrypted environment variables, and managing encryption keys.
package commands

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

// Runtime contains shared configuration and provides lazy loading for commands
type Runtime struct {
	configPath string
	keyPath    string
	Logger     zerolog.Logger
	verbose    bool

	config         *config.Config
	identity       *core.Identity
	identityLoaded bool
}

// NewRuntime creates a new context with configured logger
func NewRuntime(configPath, keyPath string, verbose bool) (*Runtime, error) {
	logger := setupLogger(verbose)

	return &Runtime{
		configPath: configPath,
		keyPath:    keyPath,
		Logger:     logger,
		verbose:    verbose,
	}, nil
}

// Config returns the configuration, loading it on first access
func (rt *Runtime) Config() (*config.Config, error) {
	if rt.config != nil {
		return rt.config, nil
	}

	// Check if config file exists before attempting to load
	if !core.FileExists(rt.configPath) {
		return nil, fmt.Errorf("configuration file '%s' not found (use 'kiln init config' to create)", rt.configPath)
	}

	cfg, err := config.Load(rt.configPath)
	if err != nil {
		return nil, fmt.Errorf("load configuration from '%s': %w", rt.configPath, err)
	}

	// Validate configuration after loading
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	rt.config = cfg
	rt.Logger.Debug().Str("config", rt.configPath).Int("recipients", len(cfg.Recipients)).Msg("configuration loaded")

	return cfg, nil
}

// Identity returns the loaded identity
func (rt *Runtime) Identity() (*core.Identity, error) {
	if rt.identityLoaded {
		return rt.identity, nil
	}

	keyPath := rt.keyPath
	if keyPath == "" {
		var err error

		keyPath, err = rt.discoverCompatibleKey()
		if err != nil {
			return nil, err
		}
	}

	identity, err := core.NewIdentityFromKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load identity from '%s': %w", keyPath, err)
	}

	rt.identity = identity
	rt.identityLoaded = true

	rt.Logger.Debug().Str("key", keyPath).Str("type", identity.KeyType()).Msg("identity loaded")

	return identity, nil
}

// Context returns a context for command operations
func (rt *Runtime) Context() context.Context {
	return context.Background()
}

// Cleanup wipes sensitive data from memory
func (rt *Runtime) Cleanup() {
	if rt.identityLoaded && rt.identity != nil {
		rt.identity.Cleanup()
		rt.identity = nil
		rt.identityLoaded = false
	}

	runtime.GC()
}

// ConfigPath returns the configuration file path
func (rt *Runtime) ConfigPath() string {
	return rt.configPath
}

func setupLogger(verbose bool) zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:          os.Stderr,
		PartsExclude: []string{zerolog.TimestampFieldName},
		FormatLevel: func(i any) string {
			levelStr, ok := i.(string)
			if !ok {
				return "UNKNOWN:"
			}

			level, err := zerolog.ParseLevel(levelStr)
			if err != nil {
				return "UNKNOWN:"
			}

			if level == zerolog.InfoLevel {
				return ""
			}

			return fmt.Sprintf("%s:", levelStr)
		},
	}

	level := zerolog.InfoLevel
	if verbose {
		level = zerolog.DebugLevel
	}

	return zerolog.New(output).Level(level)
}

func (rt *Runtime) discoverCompatibleKey() (string, error) {
	cfg, err := rt.Config()
	if err != nil {
		// No config, use default discovery
		keyPath := core.GetDefaultKeyPath()
		if keyPath == "" {
			return "", fmt.Errorf("no private key found (use 'kiln init key' or specify with --key)")
		}

		return keyPath, nil
	}

	keyPath, err := core.FindPrivateKeyForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("no compatible private key found (ensure you have a key matching the config recipients)")
	}

	return keyPath, nil
}
