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

// grpcAddress is the host:port address of the mailman gRPC server, used by raw
// gRPC clients in validation tests.
var grpcAddress string

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

	grpcAddress = fmt.Sprintf("%s:%s", host, port.Port())

	client, err := sdk.NewGRPCClient(grpcAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return client, nil
}
