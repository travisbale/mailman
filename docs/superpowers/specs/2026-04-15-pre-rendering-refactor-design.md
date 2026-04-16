# Pre-Rendering Refactor Design

## Overview

Move template rendering from the email client (async, in River worker) to the domain service (sync, at API request time). This enables fail-fast error handling: template not found, missing variables, and rendering errors are returned immediately to the gRPC client instead of failing silently in the background worker.

## Current Flow

```
gRPC handler → jobQueue.EnqueueEmailJob(JobArgs{templateName, variables})
                    ↓ (async, in River worker)
              client.Send(Email{templateName, variables})
                    ↓
              renderer.Render(templateName, variables)  ← errors lost here
```

## New Flow

```
gRPC handler → emailService.Send(SendRequest{templateName, to, variables, ...})
                    ↓ (synchronous)
              1. Load template from DB → fail if not found
              2. Validate required variables → fail if missing
              3. Render template → fail if rendering error
              4. Enqueue pre-rendered content to River
                    ↓ (async, in River worker)
              client.Send(JobArgs{subject, htmlBody, textBody, ...})  ← worker just delivers
```

## Components

### EmailService (email package)

New service that orchestrates validate -> render -> enqueue.

```go
type EmailService struct {
    templates   templateDB
    renderer    renderer
    queue       jobQueue
    fromAddress string
    fromName    string
}

func (s *EmailService) Send(ctx context.Context, req SendRequest) error
```

Interfaces defined in the `email` package:

- `templateDB` - already exists, has `GetTemplate()`
- `renderer` - `Render(ctx, templateName, variables) (*RenderedTemplate, error)`
- `jobQueue` - `EnqueueEmailJob(ctx, *JobArgs) error`

### SendRequest (email package)

New type replacing the gRPC handler's direct construction of JobArgs:

```go
type SendRequest struct {
    To          string
    TemplateName string
    Variables   map[string]string
    Priority    int32
    Metadata    map[string]string
    ScheduledAt *time.Time
}
```

### JobArgs (email package)

Changes from unrendered to pre-rendered content:

```go
type JobArgs struct {
    To       string
    From     string
    FromName string
    Subject  string
    HTMLBody string
    TextBody string
}
```

Priority, Metadata, and ScheduledAt move to queue-level insert options rather than job args, since they're not needed by the worker.

### gRPC Server

Replaces `jobQueue` dependency with `emailService`:

```go
type emailService interface {
    Send(ctx context.Context, req email.SendRequest) error
}
```

`SendEmail` converts the protobuf request to a `SendRequest` and calls `emailService.Send()`. Errors propagate back as gRPC status errors.

### River Worker

Becomes a thin delivery layer:

```go
func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[email.JobArgs]) error {
    return w.client.Send(ctx, job.Args)
}
```

### Email Clients

Signature changes from `Send(ctx, Email)` to `Send(ctx, JobArgs)`:

- **SendGrid**: Takes `JobArgs.Subject`, `JobArgs.HTMLBody`, `JobArgs.TextBody` and builds the SendGrid message. No renderer dependency.
- **Console**: Prints `JobArgs` fields to stdout. No renderer dependency.

### App Wiring (app/server.go)

Renderer is passed to `EmailService` instead of to clients:

```go
var r renderer
if config.SendGridAPIKey != "" {
    r = html.New(templatesDB)
} else {
    r = json.New()
}

emailService := email.NewEmailService(templatesDB, r, queueClient, config.FromAddress, config.FromName)
```

Clients are constructed without a renderer:

```go
var client emailClient
if config.SendGridAPIKey != "" {
    client = sendgrid.New(config.SendGridAPIKey)
} else {
    client = console.New()
}
```

## What Gets Removed

- `Email` struct (replaced by `SendRequest` for input, `JobArgs` for pre-rendered output)
- `renderer` interface and dependency from both email clients
- Renderer-related imports from client packages

## What Stays

- All three renderers (HTML, JSON, text) remain available
- Current config logic (SendGrid key = HTML renderer, no key = JSON renderer)
- `TemplateService` for template CRUD operations (unchanged)

## Integration Test Impact

With pre-rendering, the integration tests can now verify:
- Nonexistent template returns a gRPC error to the client
- Missing required variables returns a gRPC error to the client

These should be added as new test cases after the refactor.
