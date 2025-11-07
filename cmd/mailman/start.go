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
	Usage: "Start the mailman gRPC server",
	Flags: []cli.Flag{
		GRPCAddressFlag,
		SendGridAPIKeyFlag,
		FromAddressFlag,
		FromNameFlag,
		EnvironmentFlag,
	},
	Action: func(c *cli.Context) error {
		config := &app.Config{
			DatabaseURL:    c.String("database-url"),
			GRPCAddress:    c.String("grpc-address"),
			SendGridAPIKey: c.String("sendgrid-api-key"),
			FromAddress:    c.String("from-address"),
			FromName:       c.String("from-name"),
			Environment:    app.ParseEnvironment(c.String("environment")),
		}

		server, err := app.NewServer(c.Context, config)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		ctx, cancel := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)
		defer cancel()

		group, ctx := errgroup.WithContext(ctx)

		// Start server
		group.Go(func() error {
			fmt.Printf("Starting mailman service on %s\n", config.GRPCAddress)
			return server.Start()
		})

		// Handle shutdown
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
