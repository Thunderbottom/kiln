package commands

import (
	"fmt"
	"os"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

type StatusCmd struct {
	File string `short:"f" help:"Show status for specific file"`
}

func (c *StatusCmd) Run(globals *Globals) error {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return err
	}

	globals.Logger.Info("Kiln Project Status", "config", globals.Config, "recipients", len(cfg.Recipients))

	if c.File != "" {
		return c.showFileStatus(globals, c.File)
	}

	for name := range cfg.Files {
		if err := c.showFileStatus(globals, name); err != nil {
			globals.Logger.Error(fmt.Sprintf("%v", err), "file", name)
		}
	}
	return nil
}

func (c *StatusCmd) showFileStatus(globals *Globals, fileName string) error {
	filePath, info, err := core.GetFileInfo(globals.Config, fileName)
	if os.IsNotExist(err) {
		globals.Logger.Error("file not found", "file", fileName, "path", filePath)
		return nil
	} else if err != nil {
		return err
	}

	modified := info.ModTime().Format("2006-01-02 15:04:05")
	globals.Logger.Info("file metadata", "file", fileName,
		"path", filePath, "modified", modified,
		"size", fmt.Sprintf("%.2f KB", float64(info.Size())/1024.0))

	return nil
}
