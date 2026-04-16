# Pre-Rendering Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move template rendering from the async River worker to the synchronous gRPC request path, enabling fail-fast error handling for template and variable validation.

**Architecture:** A new `EmailService` in the `email` package orchestrates validate -> render -> enqueue. The gRPC handler calls `EmailService.Send()` instead of enqueueing directly. `JobArgs` changes to carry pre-rendered content. Email clients lose their renderer dependency and become pure delivery.

**Tech Stack:** Go, River job queue, gRPC, testcontainers (integration tests)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/email/models.go` | Modify | Update `JobArgs` to pre-rendered content, add `SendRequest`, remove `Email` |
| `internal/email/service.go` | Create | `EmailService`: validate template, render, enqueue |
| `internal/queue/river/client.go` | Modify | Update `EnqueueEmailJob` signature for pre-rendered `JobArgs` + `EnqueueOpts` |
| `internal/queue/river/worker.go` | Modify | Simplify worker to pass pre-rendered content to client |
| `internal/clients/console/console.go` | Modify | Remove renderer, accept pre-rendered `JobArgs` |
| `internal/clients/sendgrid/sendgrid.go` | Modify | Remove renderer, accept pre-rendered `JobArgs` |
| `internal/api/grpc/server.go` | Modify | Replace `jobQueue` with `emailService` dependency |
| `internal/api/grpc/email.go` | Modify | Call `emailService.Send()` instead of building `JobArgs` |
| `internal/app/server.go` | Modify | Wire renderer to `EmailService`, simplify client construction |
| `test/email_test.go` | Modify | Add tests for nonexistent template and missing variables |
| `test/validation_test.go` | Modify | Add raw gRPC tests for template/variable validation errors |

---

### Task 1: Update models

**Files:**
- Modify: `internal/email/models.go`

- [ ] **Step 1: Replace `Email` with `SendRequest` and update `JobArgs`**

Replace the entire contents of `internal/email/models.go` with:

```go
package email

import "time"

// SendRequest represents a request to send an email, before rendering.
type SendRequest struct {
	To           string
	TemplateName string
	Variables    map[string]string
	Priority     int32
	Metadata     map[string]string
	ScheduledAt  *time.Time
}

// Template represents an email template stored in the database
type Template struct {
	Name              string
	Subject           string
	HTMLBody          string
	TextBody          *string
	BaseTemplateName  *string
	RequiredVariables []string
	Version           int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EnqueueOpts holds queue-level options separate from the job payload.
type EnqueueOpts struct {
	Priority    int32
	ScheduledAt *time.Time
}

// JobArgs holds pre-rendered email content for the River worker.
type JobArgs struct {
	To       string
	From     string
	FromName string
	Subject  string
	HTMLBody string
	TextBody string
}

// Kind returns the unique identifier for this job type
func (JobArgs) Kind() string { return "send_email" }

// RenderedTemplate contains the rendered email content
type RenderedTemplate struct {
	Subject  string
	HTMLBody string
	TextBody string
}
```

- [ ] **Step 2: Verify the change compiles in isolation**

Run: `go build ./internal/email/...`

Expected: FAIL — other packages still reference `Email` and old `JobArgs` fields. This is expected; we'll fix them in subsequent tasks.

- [ ] **Step 3: Commit**

```bash
git add internal/email/models.go
git commit -m "Update models for pre-rendering: add SendRequest, EnqueueOpts, update JobArgs"
```

---

### Task 2: Create EmailService

**Files:**
- Create: `internal/email/service.go`

- [ ] **Step 1: Create `internal/email/service.go`**

```go
package email

import (
	"context"
	"fmt"
)

type renderer interface {
	Render(ctx context.Context, templateName string, variables map[string]string) (*RenderedTemplate, error)
}

type jobQueue interface {
	EnqueueEmailJob(ctx context.Context, jobArgs *JobArgs, opts EnqueueOpts) error
}

// EmailService orchestrates template validation, rendering, and job enqueueing.
type EmailService struct {
	templates   templateDB
	renderer    renderer
	queue       jobQueue
	fromAddress string
	fromName    string
}

// NewEmailService creates a new email service.
func NewEmailService(templates templateDB, renderer renderer, queue jobQueue, fromAddress, fromName string) *EmailService {
	return &EmailService{
		templates:   templates,
		renderer:    renderer,
		queue:       queue,
		fromAddress: fromAddress,
		fromName:    fromName,
	}
}

// Send validates the template, renders it, and enqueues the pre-rendered email.
func (s *EmailService) Send(ctx context.Context, req SendRequest) error {
	tmpl, err := s.templates.GetTemplate(ctx, req.TemplateName)
	if err != nil {
		return fmt.Errorf("template %q: %w", req.TemplateName, err)
	}

	// Validate required variables before rendering
	for _, required := range tmpl.RequiredVariables {
		if _, ok := req.Variables[required]; !ok {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}

	rendered, err := s.renderer.Render(ctx, req.TemplateName, req.Variables)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	jobArgs := &JobArgs{
		To:       req.To,
		From:     s.fromAddress,
		FromName: s.fromName,
		Subject:  rendered.Subject,
		HTMLBody: rendered.HTMLBody,
		TextBody: rendered.TextBody,
	}

	opts := EnqueueOpts{
		Priority:    req.Priority,
		ScheduledAt: req.ScheduledAt,
	}

	if err := s.queue.EnqueueEmailJob(ctx, jobArgs, opts); err != nil {
		return fmt.Errorf("failed to enqueue email: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: Verify the file compiles**

Run: `go build ./internal/email/...`

Expected: FAIL — `jobQueue` interface doesn't match current `EnqueueEmailJob` signature yet. This is expected.

- [ ] **Step 3: Commit**

```bash
git add internal/email/service.go
git commit -m "Add EmailService for pre-rendering orchestration"
```

---

### Task 3: Update River queue

**Files:**
- Modify: `internal/queue/river/client.go`
- Modify: `internal/queue/river/worker.go`

- [ ] **Step 1: Update `EnqueueEmailJob` to accept `EnqueueOpts`**

Replace the `EnqueueEmailJob` method in `internal/queue/river/client.go`:

```go
// EnqueueEmailJob enqueues a pre-rendered email job to the queue
func (c *JobQueue) EnqueueEmailJob(ctx context.Context, jobArgs *email.JobArgs, opts email.EnqueueOpts) error {
	insertOpts := &river.InsertOpts{
		MaxAttempts: 4, // Retries handle transient SendGrid API failures
		Queue:       river.QueueDefault,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // Prevents sending duplicate emails if client retries request
		},
	}

	if opts.ScheduledAt != nil {
		insertOpts.ScheduledAt = *opts.ScheduledAt
	}

	_, err := c.client.Insert(ctx, jobArgs, insertOpts)
	if err != nil {
		return fmt.Errorf("failed to enqueue email job: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: Simplify the worker**

Replace the contents of `internal/queue/river/worker.go`:

```go
package river

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/travisbale/mailman/internal/email"
)

// emailClient defines the interface for delivering pre-rendered emails
type emailClient interface {
	Send(ctx context.Context, args email.JobArgs) error
}

// SendEmailWorker processes email sending jobs from the River queue
type SendEmailWorker struct {
	river.WorkerDefaults[email.JobArgs]
	client emailClient
}

// NewSendEmailWorker creates a new email worker
func NewSendEmailWorker(client emailClient) *SendEmailWorker {
	return &SendEmailWorker{
		client: client,
	}
}

// Work delivers a pre-rendered email via the configured client
func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[email.JobArgs]) error {
	if err := w.client.Send(ctx, job.Args); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
```

- [ ] **Step 3: Update `WorkerConfig` and `NewJobQueue`**

Replace `WorkerConfig` and update `NewJobQueue` in `internal/queue/river/client.go`:

```go
// NewJobQueue creates a new River-based job queue client
func NewJobQueue(db *postgres.DB, client emailClient) (*JobQueue, error) {
	emailWorker := NewSendEmailWorker(client)
	workers := river.NewWorkers()
	river.AddWorker(workers, emailWorker)

	riverConfig := &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 5},
		},
		Workers: workers,
		// Retain job records for debugging failed email deliveries
		CompletedJobRetentionPeriod: 7 * 24 * time.Hour,
	}

	riverClient, err := river.NewClient(riverpgxv5.New(db.Pool()), riverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	return &JobQueue{
		client: riverClient,
	}, nil
}
```

Remove the `WorkerConfig` struct entirely — it's no longer needed since `NewJobQueue` takes the client directly.

- [ ] **Step 4: Verify queue package compiles**

Run: `go build ./internal/queue/...`

Expected: PASS — the queue package should compile on its own.

- [ ] **Step 5: Commit**

```bash
git add internal/queue/river/client.go internal/queue/river/worker.go
git commit -m "Update River queue for pre-rendered JobArgs"
```

---

### Task 4: Update email clients

**Files:**
- Modify: `internal/clients/console/console.go`
- Modify: `internal/clients/sendgrid/sendgrid.go`

- [ ] **Step 1: Simplify console client**

Replace the contents of `internal/clients/console/console.go`:

```go
package console

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/travisbale/mailman/internal/email"
)

// Client implements email delivery by printing emails to stdout
type Client struct {
	mu sync.Mutex // Prevents interleaved output from concurrent workers
}

// New creates a new console email client
func New() *Client {
	return &Client{}
}

// Send prints a pre-rendered email to stdout
func (c *Client) Send(ctx context.Context, args email.JobArgs) error {
	var b strings.Builder
	b.WriteString("========================================\n")
	b.WriteString("📧 Email (Console Output)\n")
	b.WriteString("========================================\n")
	fmt.Fprintf(&b, "Sent: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "From: %s <%s>\n", args.FromName, args.From)
	fmt.Fprintf(&b, "To: %s\n", args.To)
	fmt.Fprintf(&b, "Subject: %s\n", args.Subject)
	b.WriteString("----------------------------------------\n")
	if args.HTMLBody != "" {
		b.WriteString("HTML Body:\n")
		b.WriteString(args.HTMLBody)
		b.WriteString("\n")
	}
	if args.TextBody != "" {
		b.WriteString("Text Body:\n")
		b.WriteString(args.TextBody)
		b.WriteString("\n")
	}
	b.WriteString("========================================\n")

	c.mu.Lock()
	fmt.Print(b.String())
	c.mu.Unlock()

	return nil
}
```

- [ ] **Step 2: Simplify SendGrid client**

Replace the contents of `internal/clients/sendgrid/sendgrid.go`:

```go
package sendgrid

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/travisbale/mailman/internal/email"
)

// Client implements email delivery using SendGrid's API
type Client struct {
	apiKey string
}

// New creates a new SendGrid email client
func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

// Send delivers a pre-rendered email via SendGrid
func (c *Client) Send(ctx context.Context, args email.JobArgs) error {
	fromEmail := mail.NewEmail(args.FromName, args.From)
	toEmail := mail.NewEmail("", args.To)

	message := mail.NewSingleEmail(fromEmail, args.Subject, toEmail, args.TextBody, args.HTMLBody)

	client := sendgrid.NewSendClient(c.apiKey)
	response, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid returned error status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}
```

- [ ] **Step 3: Verify client packages compile**

Run: `go build ./internal/clients/...`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/clients/console/console.go internal/clients/sendgrid/sendgrid.go
git commit -m "Simplify email clients to deliver pre-rendered content"
```

---

### Task 5: Update gRPC server

**Files:**
- Modify: `internal/api/grpc/server.go`
- Modify: `internal/api/grpc/email.go`

- [ ] **Step 1: Replace `jobQueue` with `emailService` in server.go**

Replace the contents of `internal/api/grpc/server.go`:

```go
package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc"
)

type emailService interface {
	Send(ctx context.Context, req email.SendRequest) error
}

type templatesDB interface {
	List(ctx context.Context) ([]*email.Template, error)
}

// Server implements the MailmanService gRPC service
type Server struct {
	pb.UnimplementedMailmanServiceServer
	emailService emailService
	templatesDB  templatesDB
	grpcServer   *grpc.Server
	address      string
}

// NewServer creates a new gRPC server
func NewServer(address string, emailService emailService, templatesDB templatesDB) *Server {
	grpcServer := grpc.NewServer()

	server := &Server{
		emailService: emailService,
		templatesDB:  templatesDB,
		grpcServer:   grpcServer,
		address:      address,
	}

	pb.RegisterMailmanServiceServer(grpcServer, server)

	return server
}

// ListenAndServe starts the gRPC server
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	fmt.Printf("Starting gRPC server on %s\n", s.address)
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}
```

- [ ] **Step 2: Update email.go to call emailService.Send()**

Replace the contents of `internal/api/grpc/email.go`:

```go
package grpc

import (
	"context"

	"github.com/travisbale/mailman/internal/email"
	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SendEmail validates the request, then delegates to the email service for
// template rendering and job enqueueing.
func (s *Server) SendEmail(ctx context.Context, req *pb.SendEmailRequest) (*pb.SendEmailResponse, error) {
	if err := s.validateSendEmailRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	sendReq := email.SendRequest{
		To:           req.To,
		TemplateName: req.TemplateId,
		Variables:    req.Variables,
		Priority:     req.Priority,
		Metadata:     req.Metadata,
	}

	if req.ScheduledAt != nil {
		scheduledAt := req.ScheduledAt.AsTime()
		sendReq.ScheduledAt = &scheduledAt
	}

	if err := s.emailService.Send(ctx, sendReq); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send email: %v", err)
	}

	return &pb.SendEmailResponse{}, nil
}

// SendEmailBatch enqueues multiple emails in a single request
func (s *Server) SendEmailBatch(ctx context.Context, req *pb.SendEmailBatchRequest) (*pb.SendEmailBatchResponse, error) {
	results := make([]*pb.SendEmailResponse, 0, len(req.Emails))

	for _, emailReq := range req.Emails {
		resp, err := s.SendEmail(ctx, emailReq)
		if err != nil {
			return nil, err
		}
		results = append(results, resp)
	}

	return &pb.SendEmailBatchResponse{
		Results: results,
	}, nil
}

// ListTemplates returns all available email templates
func (s *Server) ListTemplates(ctx context.Context, req *pb.ListTemplatesRequest) (*pb.ListTemplatesResponse, error) {
	templates, err := s.templatesDB.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list templates: %v", err)
	}

	pbTemplates := make([]*pb.EmailTemplate, 0, len(templates))
	for _, t := range templates {
		pbTemplates = append(pbTemplates, &pb.EmailTemplate{
			Id:                t.Name,
			Subject:           t.Subject,
			RequiredVariables: t.RequiredVariables,
			Version:           t.Version,
		})
	}

	return &pb.ListTemplatesResponse{
		Templates: pbTemplates,
	}, nil
}
```

- [ ] **Step 3: Verify gRPC package compiles**

Run: `go build ./internal/api/grpc/...`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/api/grpc/server.go internal/api/grpc/email.go
git commit -m "Update gRPC server to use EmailService"
```

---

### Task 6: Update app wiring

**Files:**
- Modify: `internal/app/server.go`

- [ ] **Step 1: Rewire app server**

Replace the contents of `internal/app/server.go`:

```go
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

	queueClient, err := river.NewJobQueue(db, emailClient)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize queue client: %w", err)
	}

	emailService := email.NewEmailService(templatesDB, emailRenderer, queueClient, config.FromAddress, config.FromName)

	httpServer := &http.Server{
		Addr:              config.HTTPAddress,
		Handler:           &rest.Router{DB: db},
		ReadHeaderTimeout: 5 * time.Second, // Prevents Slowloris attacks
	}
	grpcServer := grpc.NewServer(config.GRPCAddress, emailService, templatesDB)

	return &Server{
		config:      config,
		db:          db,
		queueClient: queueClient,
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
```

Note: This file references `river.EmailClient` and `email.Renderer` as exported types. These need to be exported from their respective packages. Add the following type aliases:

In `internal/queue/river/worker.go`, the `emailClient` interface is already defined but unexported. Export it:

```go
// EmailClient defines the interface for delivering pre-rendered emails
type EmailClient interface {
	Send(ctx context.Context, args email.JobArgs) error
}
```

Update `NewSendEmailWorker` and the `SendEmailWorker` struct to use the exported `EmailClient`.

In `internal/email/service.go`, the `renderer` interface is unexported. Export it:

```go
// Renderer defines the interface for rendering email templates.
type Renderer interface {
	Render(ctx context.Context, templateName string, variables map[string]string) (*RenderedTemplate, error)
}
```

Update `EmailService` to use the exported `Renderer`.

- [ ] **Step 2: Format and build**

Run: `make fmt && make build`

Expected: PASS — the full application compiles.

- [ ] **Step 3: Commit**

```bash
git add internal/app/server.go internal/email/service.go internal/queue/river/worker.go
git commit -m "Rewire app to use EmailService for pre-rendering"
```

---

### Task 7: Add integration tests for pre-rendering validation

**Files:**
- Modify: `test/email_test.go`
- Modify: `test/validation_test.go`

- [ ] **Step 1: Add test for nonexistent template in `test/email_test.go`**

Add after the existing tests:

```go
func TestSendEmailNonexistentTemplate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailRequest{
		TemplateID: "does_not_exist",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does_not_exist")
}
```

- [ ] **Step 2: Add test for missing required variables in `test/email_test.go`**

Add after the nonexistent template test:

```go
func TestSendEmailMissingRequiredVariables(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// multi_var_template requires both Name and Company
	req := sdk.SendEmailRequest{
		TemplateID: "multi_var_template",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Company")
}
```

- [ ] **Step 3: Add raw gRPC test for nonexistent template in `test/validation_test.go`**

Add after existing validation tests:

```go
func TestSendEmailNonexistentTemplateRaw(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	_, err := client.SendEmail(context.Background(), &pb.SendEmailRequest{
		TemplateId: "does_not_exist",
		To:         "user@example.com",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "does_not_exist")
}
```

- [ ] **Step 4: Run integration tests**

Run: `make integration`

Expected: All tests pass, including the new validation tests.

- [ ] **Step 5: Commit**

```bash
git add test/email_test.go test/validation_test.go
git commit -m "Add integration tests for pre-rendering validation"
```

---

### Task 8: Clean up and verify

- [ ] **Step 1: Remove the `TemplateService` if it's now redundant**

Check if `TemplateService` in `internal/email/template.go` is still used. The `EmailService` now handles template fetching directly via the `templateDB` interface. If `TemplateService` is only used for `GetTemplate`, `CreateTemplate`, and `ListTemplates`, and those callers can use `templateDB` or `EmailService` instead, consider whether to keep it. The CLI template commands (`cmd/mailman/template.go`) likely still use `TemplateService.CreateTemplate()` for circular reference validation, so keep `TemplateService` for now.

- [ ] **Step 2: Run full test suite**

Run: `make fmt && make build && make integration`

Expected: Clean format, successful build, all integration tests pass.

- [ ] **Step 3: Final commit if any formatting changes**

Only if `make fmt` modified files.
