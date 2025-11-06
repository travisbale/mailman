package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// versionCmd is the CLI command for printing version information
var versionCmd = &cli.Command{
	Name:  "version",
	Usage: "Print version information",
	Action: func(c *cli.Context) error {
		fmt.Println("mailman version 0.1.0")
		return nil
	},
}
