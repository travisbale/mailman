# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mailman is an internal email service built with Go that provides a gRPC API for sending templated emails. It uses River (backed by PostgreSQL) for job queueing and supports both SendGrid (production) and console output (development) for email delivery.

## Architecture

### Core Components

1. **gRPC API Server** (`internal/api/grpc/`): Exposes the MailmanService gRPC API defined in `proto/mailman.proto`. The API allows clients to send single/batch emails and list available templates.

2. **River Job Queue** (`internal/queue/river/`): Background job processor that handles email sending asynchronously. Workers pull jobs from PostgreSQL and send pre-rendered emails via the configured client.

3. **Template System** (`internal/email/`): Renders email templates using Go's `html/template` package. Templates are stored in PostgreSQL with versioning and required variable validation.

4. **Email Clients** (`internal/clients/`):
   - `sendgrid/`: Production email delivery via SendGrid API
   - `console/`: Development mode that prints emails to stdout

5. **Database Layer** (`internal/db/postgres/`): PostgreSQL database using pgx/v5 driver. Code generation via sqlc for type-safe queries.

### Application Flow

**Email sending is fail-fast with pre-rendering:**

1. Client sends gRPC request → API Server loads template from DB
2. API validates required variables and renders template immediately
3. Pre-rendered content (subject, HTML, text) is enqueued to River
4. River worker pulls job → Sends email via SendGrid/Console → Job completes

**Key benefit**: Template rendering errors are returned immediately to the client, not discovered later in the worker. The worker only needs to send already-rendered content.

### Configuration

The service uses environment-based configuration with sensible defaults:

- `DATABASE_URL`: PostgreSQL connection string
- `GRPC_ADDRESS`: gRPC server bind address (default `:50051`)
- `SENDGRID_API_KEY`: When present, uses SendGrid client with HTML renderer. When absent, falls back to console client with JSON renderer.

## Development Commands

### Building

```bash
make dev          # Build with debug symbols (fast)
make build        # Build production binary (optimized)
```

### Testing

```bash
make test         # Run all tests with race detector
go test ./...     # Run tests without race detector
```

### Code Quality

```bash
make fmt          # Format code with gofmt and goimports
make lint         # Run golangci-lint (v2.6.0 via Docker)
```

The project uses golangci-lint with the "standard" preset, configured in `.golangci.yaml`. Generated code directories (`internal/pb`, `internal/db/postgres/sqlc`) are automatically excluded from linting.

### Code Generation

```bash
make protoc       # Generate protobuf/gRPC code from proto/mailman.proto
make sqlc         # Generate database code from SQL queries
```

Both `make protoc` and `make sqlc` use Docker containers to ensure consistent tooling.

### Running the Service

```bash
# Build first
make dev

# Start server (requires PostgreSQL)
./bin/mailman start --database-url="postgres://..." --sendgrid-api-key="..."

# Or use environment variables
export DATABASE_URL="postgres://..."
export SENDGRID_API_KEY="..."
./bin/mailman start
```

### Template Management

Templates can be managed via CLI commands:

```bash
# Add a simple template from files
./bin/mailman template add \
  --name welcome_email \
  --subject "Welcome {{.UserName}}!" \
  --html-file templates/welcome.html \
  --text-file templates/welcome.txt \
  --vars "UserName"

# Text file is optional if you only want HTML email
./bin/mailman template add \
  --name password_reset \
  --subject "Reset your password" \
  --html-file templates/reset.html \
  --vars "ResetLink,UserName"

# List all templates
./bin/mailman template list
```

Template files support Go template syntax (`{{.VariableName}}`). The `--vars` flag specifies required variables that must be provided when sending emails. The `--text-file` flag is optional - if omitted, only HTML emails will be sent.

#### Nested Templates

Templates support inheritance via the `--base` flag, allowing you to define common layouts:

```bash
# Create a base template with header/footer
./bin/mailman template add \
  --name company_base \
  --subject "" \
  --html-file templates/base.html

# Create content template that inherits from base
./bin/mailman template add \
  --name welcome_email \
  --subject "Welcome!" \
  --html-file templates/welcome-content.html \
  --base company_base \
  --vars "UserName,CompanyName"
```

**Base template** (`templates/base.html`):

```html
<html>
<body>
  <header>{{.CompanyName}}</header>
  {{template "content" .}}
  <footer>&copy; 2025</footer>
</body>
</html>
```

**Content template** (`templates/welcome-content.html`):

```html
{{define "content"}}
<h1>Welcome {{.UserName}}!</h1>
{{end}}
```

Templates can be nested multiple levels deep. The system provides:

- **Creation-time validation**: Circular references are detected in the TemplateService when creating templates
- **Runtime safety**: The rendering system also checks for circular references as a safety net
- **Automatic composition**: Uses Go's native `{{template}}` directive for template inheritance
- **Service layer enforcement**: Business logic lives in TemplateService, not in CLI or API layers

## SDK

The `sdk/` directory contains a public Go SDK for client applications to interact with Mailman:

- **grpc_client.go**: Wraps the generated protobuf client with a cleaner API
- **types.go**: Go-native request/response types with validation
- **README.md**: Complete SDK documentation and examples

The SDK provides:

- Type-safe, idiomatic Go API (no protobuf types exposed)
- Automatic request validation
- Functional options pattern for configuration
- Support for all Mailman operations (send, batch, list templates, etc.)

## Database

### Schema Management

Database migrations are in `internal/db/postgres/migrations/`. There is no need to create new migrations at this point - the existing migration can be edited directly as the software hasn't been released yet.

### Query Generation

SQLC generates type-safe Go code from SQL queries:

- Schema: `internal/db/postgres/migrations/*.sql`
- Queries: `internal/db/postgres/queries/*.sql`
- Generated: `internal/db/postgres/sqlc/`

After modifying queries or schema, run `make sqlc` to regenerate code.

## Protocol Buffers

The gRPC service contract is defined in `proto/mailman.proto`. After changes, regenerate with `make protoc`. Generated files go to `internal/pb/`.

## Application Lifecycle

The application follows a clean separation between CLI and application concerns:

- **CLI Layer** (`cmd/mailman/start.go`): Handles process lifecycle, signal handling (SIGTERM/SIGINT), and coordinated shutdown using `errgroup`
- **App Layer** (`internal/app/server.go`): Manages service startup and shutdown (River workers, gRPC server)

This pattern ensures:

1. Clean separation of concerns
2. 10-second graceful shutdown timeout
3. Coordinated goroutine management
4. Testability of app layer without signal handling

## Important Notes

- **No backwards compatibility concerns**: Software hasn't been released yet
- **Email templates**: Stored in `email_templates` table, use Go template syntax (`{{.Variable}}`)
- **Job queue**: River uses PostgreSQL for both queue storage and locking
- **Pre-rendering**: Templates are rendered before queueing for fail-fast error handling
- **Graceful shutdown**: CLI handles SIGTERM/SIGINT with 10-second timeout for cleanup
