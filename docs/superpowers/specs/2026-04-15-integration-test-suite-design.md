# Integration Test Suite Design

## Overview

Add an integration test suite to mailman that exercises the full gRPC API pipeline: client request -> gRPC handler -> River job queue -> email delivery (console client). Tests run against a containerized mailman instance backed by a Postgres testcontainer.

## Decisions

- **Testcontainers** over docker-compose: mailman only needs Postgres, and testcontainers keeps `go test` self-contained
- **Containerized app** over in-process: tests the real binary, and `internal/` can't be imported from `test/`
- **Flat `test/` package** within the existing module: no subdirectories, no separate `go.mod`
- **SDK client** for happy paths, raw gRPC for server-side validation edge cases
- **Direct SQL** for template seeding: decoupled from CLI, simple, works regardless of future REST API changes
- **No SendGrid key**: falls back to console client + JSON renderer

## Structure

```
test/
  main_test.go       - TestMain: container lifecycle, template seeding, SDK client setup
  setup.go           - Testcontainer helpers, SQL seeding, client creation
  email_test.go      - SendEmail, SendEmailBatch tests
  template_test.go   - ListTemplates tests
  validation_test.go - Server-side validation via raw gRPC requests
```

## TestMain Flow

1. Start Postgres testcontainer (postgres:16-alpine)
2. Seed templates via direct SQL insert into `email_templates` table
3. Build mailman Docker image from repo root `Dockerfile`
4. Start mailman container connected to the Postgres testcontainer's network, configured with:
   - `DATABASE_URL` pointing to the Postgres container
   - No `SENDGRID_API_KEY` (uses console client + JSON renderer)
   - `GRPC_ADDRESS=:50051`
5. Wait for gRPC port to accept connections
6. Create SDK client pointing at the mailman container's mapped gRPC port
7. Run all tests
8. Teardown: stop containers (handled by testcontainers cleanup)

## Seeded Templates

| Name | Subject | HTML Body | Base | Required Variables |
|------|---------|-----------|------|--------------------|
| `simple_template` | `Hello {{.Name}}!` | `<p>Welcome, {{.Name}}!</p>` | none | `Name` |
| `base_layout` | (empty) | `<html><body><header>Header</header>{{template "content" .}}<footer>Footer</footer></body></html>` | none | none |
| `nested_template` | `Nested Hello {{.Name}}!` | `{{define "content"}}<h1>Hi {{.Name}}!</h1>{{end}}` | `base_layout` | `Name` |
| `multi_var_template` | `Welcome {{.Name}} from {{.Company}}!` | `<p>{{.Name}} at {{.Company}}</p>` | none | `Name`, `Company` |

## Test Cases

### email_test.go

- **TestSendEmail**: Send email with `simple_template`, valid variables -> no error
- **TestSendEmailBatch**: Send batch with multiple recipients using `simple_template` -> no error, response contains correct number of results

### template_test.go

- **TestListTemplates**: List all templates -> returns all 4 seeded templates with correct ID, subject, required variables, and version

### validation_test.go (raw gRPC)

- **TestSendEmailMissingTemplateID**: Empty `template_id` -> `InvalidArgument`
- **TestSendEmailMissingRecipient**: Empty `to` -> `InvalidArgument`
- **TestSendEmailBatchEmpty**: Empty emails list -> error

## Verification Strategy

A successful gRPC response (no error) confirms the email was enqueued and processed by River. The console client writes JSON-rendered output to stdout inside the container. For the initial suite, we rely on the gRPC response status. Container log parsing can be added later if we need to verify rendered content.

## Dependencies to Add

- `github.com/testcontainers/testcontainers-go`
- `github.com/stretchr/testify` (assert/require)
