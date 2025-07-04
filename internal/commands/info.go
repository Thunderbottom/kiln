package commands

import (
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
	kerrors "github.com/thunderbottom/kiln/internal/errors"
)

// InfoCmd represents the info command for displaying project and file information.
type InfoCmd struct {
	File   string `short:"f" help:"Show info for specific file"`
	Verify bool   `help:"Verify file decryption capability" default:"false"`
}

func (c *InfoCmd) validate() error {
	if c.File != "" && !core.IsValidFileName(c.File) {
		return kerrors.ValidationError("file name", "cannot contain '..' or '/' characters")
	}

	return nil
}

// Run executes the info command, showing file status and verification details.
func (c *InfoCmd) Run(rt *Runtime) error {
	rt.Logger.Debug().Str("command", "info").Str("file", c.File).Bool("verify", c.Verify).Msg("validation started")

	if err := c.validate(); err != nil {
		rt.Logger.Warn().Err(err).Msg("validation failed")

		return err
	}

	cfg, err := rt.Config()
	if err != nil {
		return err
	}

	var filesToCheck []string
	if c.File != "" {
		filesToCheck = []string{c.File}
	} else {
		for name := range cfg.Files {
			filesToCheck = append(filesToCheck, name)
		}
	}

	failed := 0

	for _, fileName := range filesToCheck {
		if err := c.showFileInfo(rt, cfg, fileName); err != nil {
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("failed to get info for %d files", failed)
	}

	return nil
}

func (c *InfoCmd) showFileInfo(rt *Runtime, cfg *config.Config, fileName string) error {
	filePath, err := cfg.GetEnvFile(fileName)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		fmt.Printf("%s (%s): file not found (will be created on first use)\n", fileName, filePath)

		return nil
	} else if err != nil {
		return err
	}

	modifiedTime := fileInfo.ModTime().Format("2006-01-02 15:04:05")
	fileSizeKB := float64(fileInfo.Size()) / 1024.0

	status := c.getVerificationStatus(rt, cfg, fileName)

	fmt.Printf("%s (%s): %.2f KB, modified %s%s\n",
		fileName, filePath, fileSizeKB, modifiedTime, status)

	return nil
}

func (c *InfoCmd) getVerificationStatus(rt *Runtime, cfg *config.Config, fileName string) string {
	if !c.Verify {
		return ""
	}

	identity, err := rt.Identity()
	if err != nil {
		return " (cannot load key for verification)"
	}

	if err := core.CheckEnvFile(identity, cfg, fileName); err != nil {
		return " (cannot decrypt)"
	}

	rt.Logger.Debug().Str("file", fileName).Msg("file verification passed")

	return " (can decrypt)"
}
