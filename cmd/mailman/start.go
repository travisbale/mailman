package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/travisbale/mailman/internal/app"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

// startCmd returns the CLI command for starting the mailman server
var startCmd = &cli.Command{
	Name:  "start",
	Usage: "Start the mailman HTTP and gRPC servers",
	Flags: []cli.Flag{
		HTTPAddressFlag,
		GRPCAddressFlag,
		SendGridAPIKeyFlag,
		FromAddressFlag,
		FromNameFlag,
		EnvironmentFlag,
	},
	Action: func(c *cli.Context) error {
		appConfig := config.ToAppConfig()

		server, err := app.NewServer(c.Context, appConfig)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		ctx, cancel := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)
		defer cancel()

		group, ctx := errgroup.WithContext(ctx)

		group.Go(func() error {
			fmt.Printf("Starting mailman service (HTTP: %s, gRPC: %s)\n", appConfig.HTTPAddress, appConfig.GRPCAddress)
			return server.Start(ctx)
		})

		group.Go(func() error {
			<-ctx.Done()
			fmt.Println("Shutting down gracefully...")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			return server.Shutdown(shutdownCtx)
		})

		if err := group.Wait(); err != nil && err != context.Canceled {
			return err
		}

		return nil
	},
}
