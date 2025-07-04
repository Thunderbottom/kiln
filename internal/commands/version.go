package commands

import "fmt"

type VersionCmd struct{}

func (c *VersionCmd) Run(globals *Globals) error {
	fmt.Printf("kiln version information\n")
	return nil
}
