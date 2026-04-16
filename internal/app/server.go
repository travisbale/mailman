package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/travisbale/mailman/internal/api/grpc"
	"github.com/travisbale/mailman/internal/api/rest"
	"github.com/travisbale/mailman/internal/clients/console"
	"github.com/travisbale/mailman/internal/clients/sendgrid"
	"github.com/travisbale/mailman/internal/db/postgres"
	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/queue/river"
	"github.com/travisbale/mailman/internal/renderers/html"
	"github.com/travisbale/mailman/internal/renderers/json"
	"golang.org/x/sync/errgroup"
)

// Config holds application configuration
type Config struct {
	DatabaseURL    string
	HTTPAddress    string
	GRPCAddress    string
	SendGridAPIKey string
	FromAddress    string
	FromName       string
}

// Server represents the mailman application
type Server struct {
	config      *Config
	db          *postgres.DB
	queueClient *river.JobQueue
	httpServer  *http.Server
	grpcServer  *grpc.Server
}

// NewServer creates and initializes a new application
func NewServer(ctx context.Context, config *Config) (*Server, error) {
	db, err := postgres.NewDB(ctx, config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := postgres.MigrateUp(config.DatabaseURL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	migrator, err := rivermigrate.New(riverpgxv5.New(db.Pool()), nil)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create river migrator: %w", err)
	}
	if _, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{}); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run River migrations: %w", err)
	}

	templatesDB := postgres.NewTemplatesDB(db)

	// Select email client and renderer based on configuration
	var emailClient river.EmailClient
	var emailRenderer email.Renderer

	if config.SendGridAPIKey != "" {
		fmt.Println("Using SendGrid email client with HTML renderer")
		emailClient = sendgrid.New(config.SendGridAPIKey)
		emailRenderer = html.New(templatesDB)
	} else {
		fmt.Println("Using console email client with JSON renderer")
		emailClient = console.New()
		emailRenderer = json.New()
	}

	jobQueue, err := river.NewJobQueue(db, emailClient)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize queue client: %w", err)
	}

	emailService := &email.Service{
		Templates:   templatesDB,
		Renderer:    emailRenderer,
		Queue:       jobQueue,
		FromAddress: config.FromAddress,
		FromName:    config.FromName,
	}

	httpServer := &http.Server{
		Addr:              config.HTTPAddress,
		Handler:           &rest.Router{DB: db},
		ReadHeaderTimeout: 5 * time.Second, // Prevents Slowloris attacks
	}
	grpcServer := grpc.NewServer(config.GRPCAddress, emailService, templatesDB)

	return &Server{
		config:      config,
		db:          db,
		queueClient: jobQueue,
		httpServer:  httpServer,
		grpcServer:  grpcServer,
	}, nil
}

// Start starts the application services
func (s *Server) Start(ctx context.Context) error {
	if err := s.queueClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start River workers: %w", err)
	}

	// Start both HTTP and gRPC servers concurrently
	group, _ := errgroup.WithContext(ctx)
	group.Go(func() error { return s.httpServer.ListenAndServe() })
	group.Go(func() error { return s.grpcServer.ListenAndServe() })

	return group.Wait()
}

// Shutdown gracefully shuts down the application
func (s *Server) Shutdown(ctx context.Context) error {
	// Shutdown all services concurrently
	group, gctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		fmt.Println("Stopping HTTP server...")
		return s.httpServer.Shutdown(gctx)
	})

	group.Go(func() error {
		fmt.Println("Stopping gRPC server...")
		s.grpcServer.Stop()
		return nil
	})

	group.Go(func() error {
		fmt.Println("Stopping queue client...")
		return s.queueClient.Stop(gctx)
	})

	err := group.Wait()

	fmt.Println("Closing database connection...")
	s.db.Close()

	fmt.Println("Shutdown complete")
	return err
}
