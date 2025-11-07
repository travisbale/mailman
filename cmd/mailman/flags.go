package main

import (
	"github.com/urfave/cli/v2"
)

// Common flags that can be reused across commands
var (
	// DebugFlag enables debug logging (global flag)
	DebugFlag = &cli.BoolFlag{
		Name:    "debug",
		Usage:   "Enable debug logging",
		EnvVars: []string{"DEBUG"},
	}

	// DatabaseURLFlag defines the PostgreSQL connection URL (global flag)
	DatabaseURLFlag = &cli.StringFlag{
		Name:    "database-url",
		Usage:   "PostgreSQL connection string",
		EnvVars: []string{"DATABASE_URL"},
		Value:   "postgres://postgres:secure_password@postgres:5432/mailman?sslmode=disable",
	}

	// GRPCAddressFlag defines the gRPC server bind address
	GRPCAddressFlag = &cli.StringFlag{
		Name:    "grpc-address",
		Usage:   "gRPC server bind address",
		EnvVars: []string{"GRPC_ADDRESS"},
		Value:   ":50051",
	}

	// SendGridAPIKeyFlag defines the SendGrid API key for sending emails
	SendGridAPIKeyFlag = &cli.StringFlag{
		Name:    "sendgrid-api-key",
		Usage:   "SendGrid API key for sending emails",
		EnvVars: []string{"SENDGRID_API_KEY"},
	}

	// FromAddressFlag defines the from email address
	FromAddressFlag = &cli.StringFlag{
		Name:    "from-address",
		Usage:   "From email address",
		EnvVars: []string{"FROM_ADDRESS"},
		Value:   "no-reply@example.com",
	}

	// FromNameFlag defines the from name
	FromNameFlag = &cli.StringFlag{
		Name:    "from-name",
		Usage:   "From name",
		EnvVars: []string{"FROM_NAME"},
		Value:   "Mailman",
	}

	// EnvironmentFlag defines the environment (development/production)
	EnvironmentFlag = &cli.StringFlag{
		Name:    "environment",
		Usage:   "Environment (development/production)",
		EnvVars: []string{"ENVIRONMENT"},
		Value:   "development",
	}
)
