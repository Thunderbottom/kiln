package commands

import (
	"encoding/json"
	"fmt"
	"os"
)

type GetCmd struct {
	Key    string `arg:"" help:"Environment variable key"`
	File   string `short:"f" help:"Environment file to read from" default:"default"`
	Format string `help:"Output format" enum:"value,json" default:"value"`
}

func (c *GetCmd) Run(globals *Globals) error {
	envVars, err := loadEnvVars(globals, c.File)
	if err != nil {
		return err
	}

	value, exists := envVars[c.Key]
	if !exists {
		return fmt.Errorf("variable %s not found", c.Key)
	}

	return c.outputValue(value)
}

func (c *GetCmd) outputValue(value string) error {
	switch c.Format {
	case "value":
		fmt.Println(value)
	case "json":
		result := map[string]string{c.Key: value}
		encoder := json.NewEncoder(os.Stdout)
		return encoder.Encode(result)
	default:
		return fmt.Errorf("unsupported format: %s", c.Format)
	}

	return nil
}
