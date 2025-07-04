package commands

import (
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
)

type StatusCmd struct {
	File string `short:"f" help:"Show status for specific file"`
}

func (c *StatusCmd) Run(globals *Globals) error {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Printf("Kiln Project Status\n")
	fmt.Printf("Config file: %s\n", globals.Config)
	fmt.Printf("Recipients: %d\n", len(cfg.Recipients))

	if globals.Verbose {
		for i, recipient := range cfg.Recipients {
			fmt.Printf("  %d. %s\n", i+1, recipient)
		}
	}

	if c.File != "" {
		return c.showFileStatus(cfg, c.File)
	}

	for name := range cfg.Files {
		if err := c.showFileStatus(cfg, name); err != nil {
			fmt.Printf("  %s: Error - %v\n", name, err)
		}
	}

	return nil
}

func (c *StatusCmd) showFileStatus(cfg *config.Config, fileName string) error {
	filePath := cfg.GetEnvFile(fileName)

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		fmt.Printf("  %s (%s): File not found\n", fileName, filePath)
		return nil
	} else if err != nil {
		fmt.Printf("  %s (%s): Error - %v\n", fileName, filePath, err)
		return err
	}

	fmt.Printf("  %s (%s): %s (%d bytes)\n",
		fileName, filePath, info.ModTime().Format("2006-01-02 15:04:05"), info.Size())

	return nil
}
