# Integration Test Suite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an integration test suite that exercises the mailman gRPC API end-to-end against a containerized instance backed by a Postgres testcontainer.

**Architecture:** Tests live in `test/` as a flat package within the existing Go module. `TestMain` starts a Postgres testcontainer, seeds templates via SQL, builds and runs the mailman Docker image, then creates an SDK client. All tests use `t.Parallel()` for concurrent execution.

**Tech Stack:** testcontainers-go, testify, mailman SDK, raw gRPC (for validation tests)

---

## File Structure

| File | Responsibility |
|------|---------------|
| `test/setup.go` | Container lifecycle (Postgres + mailman), SQL seeding, SDK client creation |
| `test/main_test.go` | `TestMain` entry point: orchestrates setup, runs tests, teardown |
| `test/email_test.go` | Tests for `SendEmail` and `SendEmailBatch` RPCs |
| `test/template_test.go` | Tests for `ListTemplates` RPC |
| `test/validation_test.go` | Server-side validation tests using raw gRPC (bypassing SDK validation) |

---

### Task 1: Add testcontainers dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add testcontainers-go dependency**

```bash
go get github.com/testcontainers/testcontainers-go
```

- [ ] **Step 2: Tidy modules**

```bash
go mod tidy
```

- [ ] **Step 3: Verify module resolves**

```bash
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "Add testcontainers-go dependency for integration tests"
```

---

### Task 2: Test infrastructure setup

**Files:**
- Create: `test/setup.go`
- Create: `test/main_test.go`

- [ ] **Step 1: Create `test/setup.go` with container helpers and SQL seeding**

This file contains all infrastructure: Postgres container, mailman container, template seeding, and the shared SDK client.

```go
package test

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/travisbale/mailman/sdk"
)

// testClient is the shared SDK client used by all tests.
var testClient *sdk.GRPCClient

const (
	dbName     = "mailman"
	dbUser     = "postgres"
	dbPassword = "test_password"
)

// postgresContainer starts a PostgreSQL testcontainer and returns the container
// along with the connection string accessible from the host and from within the
// Docker network.
func postgresContainer(ctx context.Context) (testcontainers.Container, string, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       dbName,
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPassword,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to start postgres container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get postgres host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get postgres port: %w", err)
	}

	hostDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, host, port.Port(), dbName)

	// Get the container's IP on the default bridge network for container-to-container communication
	inspect, err := container.Inspect(ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to inspect postgres container: %w", err)
	}

	pgIP := inspect.NetworkSettings.Networks["bridge"].IPAddress
	internalDSN := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", dbUser, dbPassword, pgIP, dbName)

	return container, hostDSN, internalDSN, nil
}

// seedTemplates inserts test templates directly into the database.
func seedTemplates(ctx context.Context, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	// Wait for the email_templates table to exist (mailman runs migrations on startup)
	for range 30 {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = 'email_templates'
			)
		`).Scan(&exists)

		if err == nil && exists {
			break
		}
		time.Sleep(1 * time.Second)
	}

	templates := []struct {
		name             string
		subject          string
		htmlBody         string
		textBody         *string
		baseTemplateName *string
		requiredVars     string // Postgres array literal
	}{
		{
			name:         "simple_template",
			subject:      "Hello {{.Name}}!",
			htmlBody:     "<p>Welcome, {{.Name}}!</p>",
			requiredVars: "{Name}",
		},
		{
			name:         "base_layout",
			subject:      "",
			htmlBody:     `<html><body><header>Header</header>{{template "content" .}}<footer>Footer</footer></body></html>`,
			requiredVars: "{}",
		},
		{
			name:         "nested_template",
			subject:      "Nested Hello {{.Name}}!",
			htmlBody:     `{{define "content"}}<h1>Hi {{.Name}}!</h1>{{end}}`,
			requiredVars: "{Name}",
		},
		{
			name:         "multi_var_template",
			subject:      "Welcome {{.Name}} from {{.Company}}!",
			htmlBody:     "<p>{{.Name}} at {{.Company}}</p>",
			requiredVars: "{Name,Company}",
		},
	}

	// Set the base template reference for nested_template
	baseLayoutName := "base_layout"
	templates[2].baseTemplateName = &baseLayoutName

	for _, t := range templates {
		_, err := pool.Exec(ctx, `
			INSERT INTO email_templates (name, subject, html_body, text_body, base_template_name, required_variables)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (name) DO NOTHING
		`, t.name, t.subject, t.htmlBody, t.textBody, t.baseTemplateName, t.requiredVars)
		if err != nil {
			return fmt.Errorf("failed to seed template %s: %w", t.name, err)
		}
	}

	return nil
}

// mailmanContainer builds and starts the mailman Docker image, connected to the
// Postgres container via the provided internal DSN.
func mailmanContainer(ctx context.Context, internalDSN string) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "..",
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"50051/tcp"},
		Env: map[string]string{
			"DATABASE_URL": internalDSN,
			"GRPC_ADDRESS": ":50051",
			"HTTP_ADDRESS": ":8080",
			"FROM_ADDRESS": "test@example.com",
			"FROM_NAME":    "Test Mailman",
		},
		WaitingFor: wait.ForLog("Starting mailman service").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start mailman container: %w", err)
	}

	return container, nil
}

// newTestClient creates an SDK client connected to the mailman container's gRPC port.
func newTestClient(ctx context.Context, mailman testcontainers.Container) (*sdk.GRPCClient, error) {
	host, err := mailman.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mailman host: %w", err)
	}

	port, err := mailman.MappedPort(ctx, "50051")
	if err != nil {
		return nil, fmt.Errorf("failed to get mailman gRPC port: %w", err)
	}

	address := fmt.Sprintf("%s:%s", host, port.Port())

	client, err := sdk.NewGRPCClient(address)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return client, nil
}
```

- [ ] **Step 2: Create `test/main_test.go` with TestMain**

```go
package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start Postgres
	pgContainer, hostDSN, internalDSN, err := postgresContainer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres: %v\n", err)
		os.Exit(1)
	}
	defer pgContainer.Terminate(ctx)

	// Start mailman (runs migrations on startup)
	mailman, err := mailmanContainer(ctx, internalDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start mailman: %v\n", err)
		os.Exit(1)
	}
	defer mailman.Terminate(ctx)

	// Seed templates after mailman has run migrations
	if err := seedTemplates(ctx, hostDSN); err != nil {
		fmt.Fprintf(os.Stderr, "failed to seed templates: %v\n", err)
		os.Exit(1)
	}

	// Create SDK client
	testClient, err = newTestClient(ctx, mailman)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create test client: %v\n", err)
		os.Exit(1)
	}
	defer testClient.Close()

	os.Exit(m.Run())
}
```

- [ ] **Step 3: Verify the setup compiles**

```bash
go build ./test/...
```

Expected: clean build. Tests won't run yet since there are no test functions.

- [ ] **Step 4: Commit**

```bash
git add test/setup.go test/main_test.go
git commit -m "Add integration test infrastructure with testcontainers"
```

---

### Task 3: Email sending tests

**Files:**
- Create: `test/email_test.go`

- [ ] **Step 1: Create `test/email_test.go`**

```go
package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/travisbale/mailman/sdk"
)

func TestSendEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailRequest{
		TemplateID: "simple_template",
		To:         "user@example.com",
		Variables:  map[string]string{"Name": "Alice"},
	}

	_, err := testClient.SendEmail(ctx, req)
	require.NoError(t, err)
}

func TestSendEmailBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	req := sdk.SendEmailBatchRequest{
		Emails: []sdk.SendEmailRequest{
			{
				TemplateID: "simple_template",
				To:         "user1@example.com",
				Variables:  map[string]string{"Name": "Alice"},
			},
			{
				TemplateID: "simple_template",
				To:         "user2@example.com",
				Variables:  map[string]string{"Name": "Bob"},
			},
		},
	}

	resp, err := testClient.SendEmailBatch(ctx, req)
	require.NoError(t, err)
	assert.Len(t, resp.Results, 2)
}
```

- [ ] **Step 2: Run the tests**

```bash
go test -v -count=1 ./test/...
```

Expected: both tests pass. This will take a while on first run since it builds the Docker image.

- [ ] **Step 3: Commit**

```bash
git add test/email_test.go
git commit -m "Add integration tests for SendEmail and SendEmailBatch"
```

---

### Task 4: Template listing tests

**Files:**
- Create: `test/template_test.go`

- [ ] **Step 1: Create `test/template_test.go`**

```go
package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	resp, err := testClient.ListTemplates(ctx)
	require.NoError(t, err)
	assert.Len(t, resp.Templates, 4)

	// Build a map for easier assertion
	templates := make(map[string]struct {
		subject  string
		vars     []string
		version  int32
	})
	for _, tmpl := range resp.Templates {
		templates[tmpl.ID] = struct {
			subject  string
			vars     []string
			version  int32
		}{
			subject: tmpl.Subject,
			vars:    tmpl.RequiredVariables,
			version: tmpl.Version,
		}
	}

	simple := templates["simple_template"]
	assert.Equal(t, "Hello {{.Name}}!", simple.subject)
	assert.Equal(t, []string{"Name"}, simple.vars)
	assert.Equal(t, int32(1), simple.version)

	multi := templates["multi_var_template"]
	assert.Equal(t, "Welcome {{.Name}} from {{.Company}}!", multi.subject)
	assert.ElementsMatch(t, []string{"Name", "Company"}, multi.vars)
}
```

- [ ] **Step 2: Run the tests**

```bash
go test -v -count=1 ./test/...
```

Expected: all tests pass including the new template test.

- [ ] **Step 3: Commit**

```bash
git add test/template_test.go
git commit -m "Add integration test for ListTemplates"
```

---

### Task 5: Validation tests with raw gRPC

**Files:**
- Create: `test/validation_test.go`

- [ ] **Step 1: Create `test/validation_test.go`**

These tests bypass SDK-side validation by making raw gRPC calls directly, so we can verify the server rejects invalid requests.

```go
package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/travisbale/mailman/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// rawGRPCClient returns a raw protobuf client that bypasses SDK validation.
// It reuses the same address as the SDK test client by reading it from the
// mailman container at test setup time. We store the address in a package-level
// variable set during TestMain.
func rawGRPCClient(t *testing.T) pb.MailmanServiceClient {
	t.Helper()

	conn, err := grpc.NewClient(
		grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return pb.NewMailmanServiceClient(conn)
}

func TestSendEmailMissingTemplateID(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	_, err := client.SendEmail(context.Background(), &pb.SendEmailRequest{
		TemplateId: "",
		To:         "user@example.com",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "template_id")
}

func TestSendEmailMissingRecipient(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	_, err := client.SendEmail(context.Background(), &pb.SendEmailRequest{
		TemplateId: "simple_template",
		To:         "",
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "to")
}

func TestSendEmailBatchEmpty(t *testing.T) {
	t.Parallel()

	client := rawGRPCClient(t)

	resp, err := client.SendEmailBatch(context.Background(), &pb.SendEmailBatchRequest{
		Emails: []*pb.SendEmailRequest{},
	})

	// An empty batch currently succeeds with an empty results list.
	// If this behavior changes to return an error, update this test.
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}
```

- [ ] **Step 2: Add `grpcAddress` variable to `test/setup.go`**

Add the package-level variable and set it in `newTestClient`:

In `test/setup.go`, add after the `testClient` variable declaration:

```go
// grpcAddress is the host:port address of the mailman gRPC server, used by raw
// gRPC clients in validation tests.
var grpcAddress string
```

Update `newTestClient` to also set `grpcAddress`:

```go
func newTestClient(ctx context.Context, mailman testcontainers.Container) (*sdk.GRPCClient, error) {
	host, err := mailman.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get mailman host: %w", err)
	}

	port, err := mailman.MappedPort(ctx, "50051")
	if err != nil {
		return nil, fmt.Errorf("failed to get mailman gRPC port: %w", err)
	}

	grpcAddress = fmt.Sprintf("%s:%s", host, port.Port())

	client, err := sdk.NewGRPCClient(grpcAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return client, nil
}
```

- [ ] **Step 3: Run all tests**

```bash
go test -v -count=1 ./test/...
```

Expected: all tests pass. The empty batch test documents current behavior (succeeds with empty results).

- [ ] **Step 4: Commit**

```bash
git add test/validation_test.go test/setup.go
git commit -m "Add validation integration tests with raw gRPC client"
```

---

### Task 6: Run full suite and verify

- [ ] **Step 1: Run the complete test suite with race detector**

```bash
go test -v -race -count=1 ./test/...
```

Expected: all tests pass with no race conditions detected.

- [ ] **Step 2: Run `make fmt` to ensure formatting is clean**

```bash
make fmt
```

Expected: no files modified.

- [ ] **Step 3: Final commit if any formatting changes**

Only if `make fmt` modified files:

```bash
git add -A
git commit -m "Format integration test files"
```
