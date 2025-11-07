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
			if c.Bool("debug") {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				})))
			} else {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelInfo,
				})))
			}
			return nil
		},
		Commands: []*cli.Command{
			startCmd,
			templateCmd,
			versionCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
