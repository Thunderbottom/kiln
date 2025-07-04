package commands

import (
	"fmt"

	"github.com/thunderbottom/kiln/internal/config"
	"github.com/thunderbottom/kiln/internal/core"
)

type VerifyCmd struct {
	File string `short:"f" help:"Verify specific file"`
}

func (c *VerifyCmd) Run(globals *Globals) error {
	cfg, err := config.Load(globals.Config)
	if err != nil {
		return err
	}

	var filesToVerify []string
	if c.File != "" {
		filesToVerify = []string{c.File}
	} else {
		for name := range cfg.Files {
			filesToVerify = append(filesToVerify, name)
		}
	}

	successful := 0
	for _, fileName := range filesToVerify {
		if err := core.ValidateEnvFile(globals.Config, fileName); err != nil {
			globals.Logger.Info(fmt.Sprintf("%v", err), "file", fileName)
		} else {
			globals.Logger.Info("ok", "file", fileName)
			successful++
		}
	}

	globals.Logger.Info("verification complete", "success", successful, "total", len(filesToVerify))

	if successful < len(filesToVerify) {
		return fmt.Errorf("verification failed for %d file(s)", len(filesToVerify)-successful)
	}
	return nil
}
