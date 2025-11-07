package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var (
	// Version is set at build time via ldflags
	Version = "dev"
)

// versionCmd is the CLI command for printing version information
var versionCmd = &cli.Command{
	Name:  "version",
	Usage: "Print version information",
	Action: func(c *cli.Context) error {
		fmt.Printf("mailman version %s\n", Version)
		return nil
	},
}
