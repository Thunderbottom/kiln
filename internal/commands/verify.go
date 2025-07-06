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

	ctx := globals.Context()
	successful := 0
	for _, fileName := range filesToVerify {
		if err := core.CheckFile(ctx, globals.Config, fileName); err != nil {
			globals.Logger.Info().Str("file", fileName).Err(err)
		} else {
			globals.Logger.Info().Str("file", fileName).Msg("ok")
			successful++
		}
	}

	globals.Logger.Info().
		Int("success", successful).
		Int("total", len(filesToVerify)).
		Msg("verification complete")

	if successful < len(filesToVerify) {
		return fmt.Errorf("verification failed for %d file(s)", len(filesToVerify)-successful)
	}
	return nil
}
