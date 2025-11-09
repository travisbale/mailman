package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "mailman",
		Usage: "Internal email service for payroll platform",
		Flags: []cli.Flag{
			DebugFlag,
			DatabaseURLFlag,
		},
		Before: func(c *cli.Context) error {
			// Set log level based on debug flag
			var level slog.Level
			if config.Debug {
				level = slog.LevelDebug
			} else {
				level = slog.LevelInfo
			}

			opts := &slog.HandlerOptions{Level: level}
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))

			return nil
		},
		Commands: []*cli.Command{
			startCmd,
			migrateCmd,
			templateCmd,
			versionCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
