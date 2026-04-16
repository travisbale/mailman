package river

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/travisbale/mailman/internal/email"
)

// EmailClient defines the interface for delivering pre-rendered emails
type EmailClient interface {
	Send(ctx context.Context, args email.JobArgs) error
}

// SendEmailWorker processes email sending jobs from the River queue
type SendEmailWorker struct {
	river.WorkerDefaults[email.JobArgs]
	client EmailClient
}

// NewSendEmailWorker creates a new email worker
func NewSendEmailWorker(client EmailClient) *SendEmailWorker {
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
