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
