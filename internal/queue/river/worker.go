package river

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/travisbale/mailman/internal/email"
)

// emailService defines the interface for sending emails
type emailService interface {
	Send(ctx context.Context, email email.Email) error
}

// SendEmailWorker processes email sending jobs from the River queue
type SendEmailWorker struct {
	river.WorkerDefaults[email.JobArgs]
	emailService emailService
	fromAddress  string
	fromName     string
}

// NewSendEmailWorker creates a new email worker
func NewSendEmailWorker(config WorkerConfig) *SendEmailWorker {
	return &SendEmailWorker{
		emailService: config.EmailService,
		fromAddress:  config.FromAddress,
		fromName:     config.FromName,
	}
}

// Work processes a single email job by passing an Email struct to the client
func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[email.JobArgs]) error {
	args := job.Args

	// Build email with template info
	email := email.Email{
		To:           args.To,
		From:         w.fromAddress,
		FromName:     w.fromName,
		TemplateName: args.TemplateName,
		Variables:    args.Variables,
	}

	// Email client handles rendering and sending
	if err := w.emailService.Send(ctx, email); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
