# Mailman

[![CI](https://github.com/travisbale/mailman/actions/workflows/ci.yml/badge.svg)](https://github.com/travisbale/mailman/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/travisbale/mailman)](https://goreportcard.com/report/github.com/travisbale/mailman)
[![Go Version](https://img.shields.io/badge/go-1.25-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A high-performance email service built with Go that provides a gRPC API for sending templated emails at scale. Mailman uses PostgreSQL-backed job queuing with River to ensure reliable email delivery with built-in retry logic.

## Features

- **gRPC API**: Fast, type-safe communication with protocol buffers
- **Template System**: Store and version email templates in PostgreSQL with Go template syntax
- **Asynchronous Processing**: Background job queue with automatic retries
- **Multiple Backends**: SendGrid for production, console output for development
- **Job Scheduling**: Schedule emails for future delivery
- **Batch Operations**: Send multiple emails in a single request
- **Status Tracking**: Query job status and delivery history

## Prerequisites

- Go 1.24 or higher
- PostgreSQL 12 or higher
- Docker (for code generation)
- SendGrid API key (for production use)

## Installation

### As a Service

```bash
make deps
```

### As a Client Library (SDK)

```bash
go get github.com/travisbale/mailman/sdk
```

## Quick Start

### 1. Install Dependencies (Service)

```bash
make deps
```

### 2. Set Up Database

Create a PostgreSQL database and run migrations:

```bash
# Create database
createdb mailman

# Set database URL
export DATABASE_URL="postgres://postgres:password@localhost:5432/mailman?sslmode=disable"

# Migrations are applied automatically on first run
```

### 3. Build and Run

```bash
# Build development binary
make dev

# Run in development mode (emails printed to console)
export ENVIRONMENT=development
./bin/mailman start

# Or run in production mode with SendGrid
export ENVIRONMENT=production
export SENDGRID_API_KEY="your-sendgrid-api-key"
export FROM_ADDRESS="noreply@yourdomain.com"
export FROM_NAME="Your App Name"
./bin/mailman start
```

The gRPC server will start on port 50051 by default.

## Configuration

Configuration is done via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:secure_password@postgres:5432/mailman?sslmode=disable` |
| `GRPC_ADDRESS` | gRPC server bind address | `:50051` |
| `ENVIRONMENT` | Environment mode (`development`/`production`) | `development` |
| `SENDGRID_API_KEY` | SendGrid API key (required for production) | - |
| `FROM_ADDRESS` | Default from email address | `no-reply@example.com` |
| `FROM_NAME` | Default from name | `Mailman` |

## Usage

### Managing Email Templates

Templates use Go template syntax (`{{.VariableName}}`) and are stored in the `email_templates` table.

#### Add a Template via CLI

```bash
# Add a simple template
./bin/mailman template add \
  --name welcome_email \
  --subject "Welcome to {{.AppName}}, {{.UserName}}!" \
  --html-file templates/welcome.html \
  --text-file templates/welcome.txt \
  --vars "UserName,AppName"

# Text file is optional if you only want HTML email
./bin/mailman template add \
  --name password_reset \
  --subject "Reset your password" \
  --html-file templates/reset.html \
  --vars "ResetLink,UserName"

# List all templates
./bin/mailman template list
```

#### Nested Templates

Templates can inherit from a base template for consistent branding. This allows you to define headers, footers, and styling once and reuse across all emails.

**Step 1: Create a base template**
```bash
./bin/mailman template add \
  --name company_base \
  --subject "" \
  --html-file templates/base.html
```

**templates/base.html:**
```html
<!DOCTYPE html>
<html>
<head>
  <style>
    .header { background: #003366; color: white; padding: 20px; }
    .footer { background: #f0f0f0; padding: 10px; text-align: center; }
  </style>
</head>
<body>
  <div class="header">
    <h1>{{.CompanyName}}</h1>
  </div>

  <div class="content">
    {{template "content" .}}
  </div>

  <div class="footer">
    &copy; 2025 {{.CompanyName}}. All rights reserved.
  </div>
</body>
</html>
```

**Step 2: Create content templates that inherit from the base**
```bash
./bin/mailman template add \
  --name welcome_email \
  --subject "Welcome {{.UserName}}!" \
  --html-file templates/welcome-content.html \
  --base company_base \
  --vars "UserName,CompanyName"
```

**templates/welcome-content.html:**
```html
{{define "content"}}
<h2>Welcome {{.UserName}}!</h2>
<p>Thanks for joining our platform.</p>
{{end}}
```

The rendered email will include the full layout with header and footer from `company_base`. Templates can be nested multiple levels deep.

**Safety Features:**
- Circular references are validated when creating templates (fails immediately with clear error)
- The CLI verifies the entire inheritance chain before saving
- Runtime checks provide an additional safety layer

#### Add a Template via SQL

Alternatively, insert templates directly:

```sql
-- Simple template
INSERT INTO email_templates (name, subject, html_body, text_body, required_variables)
VALUES (
    'welcome_email',
    'Welcome to {{.AppName}}, {{.UserName}}!',
    '<h1>Welcome {{.UserName}}!</h1><p>Thanks for joining {{.AppName}}.</p>',
    'Welcome {{.UserName}}! Thanks for joining {{.AppName}}.',
    ARRAY['UserName', 'AppName']
);

-- Nested template with base
INSERT INTO email_templates (name, subject, html_body, base_template_name, required_variables)
VALUES (
    'password_reset',
    'Reset your password',
    '{{define "content"}}<h2>Password Reset</h2><p>Click here: {{.ResetLink}}</p>{{end}}',
    'company_base',
    ARRAY['ResetLink', 'CompanyName']
);
```

### Sending Emails via SDK

The easiest way to send emails is using the Mailman SDK:

```go
import "github.com/travisbale/mailman/sdk"

// Create client
client, err := sdk.NewGRPCClient("localhost:50051")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Send email
req := sdk.SendEmailRequest{
    TemplateID: "welcome_email",
    To:         "user@example.com",
    Variables: map[string]string{
        "UserName": "Alice",
        "AppName":  "MyApp",
    },
}

resp, err := client.SendEmail(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
```

See [sdk/README.md](sdk/README.md) for complete SDK documentation.

### Sending Emails via gRPC (Raw)

Example using the raw gRPC client:

```go
import (
    "context"
    "google.golang.org/grpc"
    "github.com/travisbale/mailman/internal/pb"
)

conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := pb.NewMailmanServiceClient(conn)

// Send single email
resp, err := client.SendEmail(context.Background(), &pb.SendEmailRequest{
    TemplateId: "welcome_email",
    To:         "user@example.com",
    Variables: map[string]string{
        "UserName": "Alice",
        "AppName":  "MyApp",
    },
    Priority: 0,
})

// Send batch emails
batchResp, err := client.SendEmailBatch(context.Background(), &pb.SendEmailBatchRequest{
    Emails: []*pb.SendEmailRequest{
        {TemplateId: "welcome_email", To: "user1@example.com", Variables: ...},
        {TemplateId: "welcome_email", To: "user2@example.com", Variables: ...},
    },
})

// List available templates
templates, err := client.ListTemplates(context.Background(), &pb.ListTemplatesRequest{})
```

### CLI Commands

```bash
# Start the server
./bin/mailman start

# Manage templates
./bin/mailman template add --name <template_name> --subject <subject> ...
./bin/mailman template list

# Show version
./bin/mailman version

# Show help
./bin/mailman --help
```

## Development

### Building

```bash
make dev          # Development build (fast, with debug symbols)
make build        # Production build (optimized)
```

### Testing

```bash
make test         # Run tests with race detector
go test ./...     # Run tests without race detector
```

### Code Quality

```bash
make fmt          # Format code with gofmt and goimports
make lint         # Run golangci-lint (v2.6.0, standard preset)
```

Linting is configured in `.golangci.yaml` with the "standard" preset. Generated code is automatically excluded.

### Code Generation

After modifying proto files or SQL queries:

```bash
make protoc       # Generate protobuf/gRPC code
make sqlc         # Generate database code from SQL
```

### Project Structure

```
mailman/
├── cmd/mailman/          # CLI entrypoint
├── internal/
│   ├── api/
│   │   └── grpc/         # gRPC server implementation
│   ├── app/              # Application setup and lifecycle
│   ├── clients/          # Email delivery clients (SendGrid, console)
│   ├── db/postgres/      # Database layer with migrations
│   ├── mail/             # Domain models and template rendering service
│   ├── pb/               # Generated protobuf code
│   └── queue/river/      # River job queue and workers
├── proto/                # Protocol buffer definitions
└── sdk/                  # Public Go SDK for client applications
```

## Architecture

Mailman follows a clean architecture pattern with fail-fast template rendering:

1. **API Layer**: gRPC server loads, validates, and renders templates before enqueueing jobs
2. **Queue Layer**: River workers send pre-rendered email content asynchronously from PostgreSQL
3. **Domain Layer**: Business logic for template rendering and validation
4. **Infrastructure Layer**: Email clients (SendGrid/console) and database

**Email Flow**: All email sending is asynchronous. The API renders templates immediately (returning any errors to the client), then enqueues the pre-rendered content. River workers handle the actual email delivery with automatic retries on failure. This fail-fast approach ensures clients receive template errors immediately rather than discovering them later in the queue.

## License

See LICENSE file.
