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
	session, err := globals.Session()
	if err != nil {
		return fmt.Errorf("initialize session: %w", err)
	}

	cfg := session.Config()

	globals.Logger.Info().
		Str("config_path", globals.Config).
		Int("recipients", len(cfg.Recipients)).
		Msg("kiln project info")

	var filesToCheck []string
	if c.File != "" {
		filesToCheck = []string{c.File}
		globals.Logger.Debug().
			Str("file", c.File).
			Msg("checking specific file")
	} else {
		for name := range cfg.Files {
			filesToCheck = append(filesToCheck, name)
		}
		globals.Logger.Debug().
			Int("file_count", len(filesToCheck)).
			Msg("checking all configured files")
	}

	successful := 0
	failed := 0

	for _, fileName := range filesToCheck {
		globals.Logger.Debug().
			Str("file", fileName).
			Bool("verify", c.Verify).
			Msg("processing file info")

		if err := c.showFileInfo(session, globals, fileName); err != nil {
			globals.Logger.Error().
				Err(err).
				Str("file", fileName).
				Msg("failed to get file info")
			failed++
		} else {
			successful++
		}
	}

	if len(filesToCheck) > 1 {
		globals.Logger.Info().
			Int("successful", successful).
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

func (c *InfoCmd) showFileInfo(session *core.Session, globals *Globals, fileName string) error {
	filePath, fileInfo, err := session.GetFileInfo(fileName)
	if os.IsNotExist(err) {
		globals.Logger.Warn().
			Str("file", fileName).
			Str("path", filePath).
			Msg("file not found")
		return nil
	} else if err != nil {
		globals.Logger.Error().
			Err(err).
			Str("file", fileName).
			Msg("failed to get file info")
		return err
	}

	modifiedTime := fileInfo.ModTime().Format("2006-01-02 15:04:05")
	fileSizeKB := float64(fileInfo.Size()) / 1024.0

	logger := globals.Logger.Info().
		Str("file", fileName).
		Str("path", filePath).
		Str("modified", modifiedTime).
		Str("size", fmt.Sprintf("%.2fKB", fileSizeKB))

	if c.Verify {
		globals.Logger.Debug().
			Str("file", fileName).
			Msg("verifying file decryption capability")

		if err := session.CheckFile(fileName); err != nil {
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
