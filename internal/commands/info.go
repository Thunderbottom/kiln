package commands

import (
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/core"
)

type InfoCmd struct {
	File   string `short:"f" help:"Show info for specific file"`
	Verify bool   `help:"Verify file decryption capability" default:"false"`
}

func (c *InfoCmd) Run(globals *Globals) error {
	sess, err := globals.Session()
	if err != nil {
		return err
	}

	cfg := sess.Config()
	globals.Logger.Info().
		Str("config", globals.Config).
		Int("recipients", len(cfg.Recipients)).
		Msg("kiln project info")

	var filesToCheck []string
	if c.File != "" {
		filesToCheck = []string{c.File}
	} else {
		for name := range cfg.Files {
			filesToCheck = append(filesToCheck, name)
		}
	}

	successful := 0
	failed := 0

	for _, fileName := range filesToCheck {
		if err := c.showFileInfo(globals, sess, fileName); err != nil {
			globals.Logger.Error().Err(err).Str("file", fileName).Msg("failed to get file info")
			failed++
		} else {
			successful++
		}
	}

	// Show summary if checking multiple files
	if len(filesToCheck) > 1 {
		globals.Logger.Info().
			Int("success", successful).
			Int("failed", failed).
			Int("total", len(filesToCheck)).
			Bool("verified", c.Verify).
			Msg("info summary")
	}

	if failed > 0 {
		return fmt.Errorf("failed to get info for %d file(s)", failed)
	}

	return nil
}

func (c *InfoCmd) showFileInfo(globals *Globals, sess *core.Session, fileName string) error {
	// Get file metadata
	filePath, info, err := sess.GetFileInfo(fileName)
	if os.IsNotExist(err) {
		globals.Logger.Warn().
			Str("file", fileName).
			Str("path", filePath).
			Msg("file not found")
		return nil
	} else if err != nil {
		return err
	}

	// Show basic file metadata
	modified := info.ModTime().Format("2006-01-02 15:04:05")
	logger := globals.Logger.Info().
		Str("file", fileName).
		Str("path", filePath).
		Str("modified", modified).
		Str("size", fmt.Sprintf("%.2fKB", float64(info.Size())/1024.0))

	// Verify decryption if requested
	if c.Verify {
		if err := sess.CheckFile(fileName); err != nil {
			logger.Str("status", "failed").
				Err(err).
				Msg("file info with verification")
			return err
		} else {
			logger.Str("status", "ok").
				Msg("file info with verification")
		}
	} else {
		logger.Msg("file info")
	}

	return nil
}
