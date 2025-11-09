package main

import (
	"github.com/travisbale/mailman/internal/app"
)

// Config holds all configuration for the application
type Config struct {
	// Debug
	Debug bool

	// Database
	DatabaseURL string

	// Server address
	GRPCAddress string

	// SendGrid configuration
	SendGridAPIKey string

	// Email sender configuration
	FromAddress string
	FromName    string

	// Environment
	Environment string
}

// config is the global configuration populated by CLI flags
var config = &Config{}

// ToAppConfig converts the CLI config to an app.Config
func (c *Config) ToAppConfig() *app.Config {
	return &app.Config{
		DatabaseURL:    c.DatabaseURL,
		GRPCAddress:    c.GRPCAddress,
		SendGridAPIKey: c.SendGridAPIKey,
		FromAddress:    c.FromAddress,
		FromName:       c.FromName,
		Environment:    app.ParseEnvironment(c.Environment),
	}
}
