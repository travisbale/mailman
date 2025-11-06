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
		&cli.StringFlag{
			Name:    "database-url",
			Usage:   "PostgreSQL connection string",
			EnvVars: []string{"DATABASE_URL"},
			Value:   "postgres://postgres:secure_password@postgres:5432/mailman?sslmode=disable",
		},
		&cli.StringFlag{
			Name:    "grpc-address",
			Usage:   "gRPC server bind address",
			EnvVars: []string{"GRPC_ADDRESS"},
			Value:   ":50051",
		},
		&cli.StringFlag{
			Name:    "sendgrid-api-key",
			Usage:   "SendGrid API key for sending emails",
			EnvVars: []string{"SENDGRID_API_KEY"},
		},
		&cli.StringFlag{
			Name:    "from-address",
			Usage:   "From email address",
			EnvVars: []string{"FROM_ADDRESS"},
			Value:   "no-reply@example.com",
		},
		&cli.StringFlag{
			Name:    "from-name",
			Usage:   "From name",
			EnvVars: []string{"FROM_NAME"},
			Value:   "Mailman",
		},
		&cli.StringFlag{
			Name:    "environment",
			Usage:   "Environment (development/production)",
			EnvVars: []string{"ENVIRONMENT"},
			Value:   "development",
		},
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
