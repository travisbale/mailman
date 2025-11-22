package river

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/travisbale/mailman/internal/db/postgres"
	"github.com/travisbale/mailman/internal/email"
)

// JobQueue wraps the River client for email job processing
type JobQueue struct {
	client *river.Client[pgx.Tx]
}

// WorkerConfig holds configuration for email workers
type WorkerConfig struct {
	EmailService emailService
	FromAddress  string
	FromName     string
}

// NewJobQueue creates a new River-based job queue client
func NewJobQueue(db *postgres.DB, config WorkerConfig) (*JobQueue, error) {
	emailWorker := NewSendEmailWorker(config)
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

// EnqueueEmailJob enqueues an email job to the queue
func (c *JobQueue) EnqueueEmailJob(ctx context.Context, jobArgs *email.JobArgs) error {
	insertOpts := &river.InsertOpts{
		MaxAttempts: 4, // Retries handle transient SendGrid API failures
		Priority:    0,
		Queue:       river.QueueDefault,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true, // Prevents sending duplicate emails if client retries request
		},
	}

	if jobArgs.ScheduledAt != nil {
		insertOpts.ScheduledAt = *jobArgs.ScheduledAt
	}

	_, err := c.client.Insert(ctx, jobArgs, insertOpts)
	if err != nil {
		return fmt.Errorf("failed to enqueue email job: %w", err)
	}

	return nil
}

// Start starts the River job queue workers
func (c *JobQueue) Start(ctx context.Context) error {
	if err := c.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start River client: %w", err)
	}
	return nil
}

// Stop stops the River client gracefully
func (c *JobQueue) Stop(ctx context.Context) error {
	if err := c.client.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop River client: %w", err)
	}
	return nil
}
