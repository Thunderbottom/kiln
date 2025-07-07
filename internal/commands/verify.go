package commands

import "fmt"

type VerifyCmd struct {
	File string `short:"f" help:"Verify specific file"`
}

func (c *VerifyCmd) Run(globals *Globals) error {
	sess, err := globals.Session()
	if err != nil {
		return err
	}

	cfg := sess.Config()
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
		if err := sess.CheckFile(ctx, fileName); err != nil {
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
