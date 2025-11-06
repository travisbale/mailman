package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/travisbale/mailman/internal/api/grpc"
	"github.com/travisbale/mailman/internal/clients/console"
	"github.com/travisbale/mailman/internal/clients/sendgrid"
	"github.com/travisbale/mailman/internal/db/postgres"
	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/queue/river"
)

// Environment represents the application environment
type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

// ParseEnvironment parses a string into an Environment type
func ParseEnvironment(s string) Environment {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "production", "prod":
		return Production
	default:
		return Development
	}
}

// Config holds application configuration
type Config struct {
	DatabaseURL    string
	GRPCAddress    string
	SendGridAPIKey string
	FromAddress    string
	FromName       string
	Environment    Environment
}

type emailSender interface {
	Send(ctx context.Context, email email.Email) error
}

// Server represents the mailman application
type Server struct {
	config      *Config
	db          *postgres.DB
	queueClient *river.JobQueue
	grpcServer  *grpc.Server
	templatesDB *postgres.TemplatesDB
}

// NewServer creates and initializes a new application
func NewServer(ctx context.Context, config *Config) (*Server, error) {
	// Initialize database
	db, err := postgres.NewDB(ctx, config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create templates DB adapter
	templatesDB := postgres.NewTemplatesDB(db)

	// Determine email sender based on environment
	var emailSender emailSender
	if config.Environment == Development || config.SendGridAPIKey == "" {
		fmt.Println("Using console email client (development mode)")
		emailSender = console.New()
	} else {
		fmt.Println("Using SendGrid email client")
		emailSender = sendgrid.New(config.SendGridAPIKey)
	}

	// Create template renderer
	templateService := email.NewTemplateService(templatesDB)

	// Initialize River queue client
	queueClient, err := river.NewJobQueue(db, river.WorkerConfig{
		EmailService: emailSender,
		FromAddress:  config.FromAddress,
		FromName:     config.FromName,
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize queue client: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(config.GRPCAddress, templateService, queueClient, templatesDB)

	return &Server{
		config:      config,
		db:          db,
		queueClient: queueClient,
		grpcServer:  grpcServer,
		templatesDB: templatesDB,
	}, nil
}

// Start starts the application services
func (s *Server) Start() error {
	// Start River workers
	if err := s.queueClient.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start River workers: %w", err)
	}

	// Start gRPC server (blocks until error or shutdown)
	if err := s.grpcServer.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the application
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("Stopping gRPC server...")
	s.grpcServer.Stop()

	fmt.Println("Stopping queue client...")
	if err := s.queueClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop queue client: %w", err)
	}

	fmt.Println("Closing database connection...")
	s.db.Close()

	fmt.Println("Shutdown complete")
	return nil
}
