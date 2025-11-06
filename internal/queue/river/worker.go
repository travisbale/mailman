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
// Email content is pre-rendered before queueing, so worker just sends
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

// Work processes a single email job
// Content is already rendered, so just send the email
func (w *SendEmailWorker) Work(ctx context.Context, job *river.Job[email.JobArgs]) error {
	args := job.Args

	// Send email via email client (content already rendered)
	email := email.Email{
		From:     w.fromAddress,
		FromName: w.fromName,
		To:       args.To,
		Subject:  args.Subject,
		HTMLBody: args.HTMLBody,
		TextBody: args.TextBody,
	}

	if err := w.emailService.Send(ctx, email); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
