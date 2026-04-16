package sendgrid

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/travisbale/mailman/internal/email"
)

// Client implements email delivery using SendGrid's API
type Client struct {
	apiKey string
}

// New creates a new SendGrid email client
func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

// Send delivers a pre-rendered email via SendGrid
func (c *Client) Send(ctx context.Context, args email.JobArgs) error {
	fromEmail := mail.NewEmail(args.FromName, args.From)
	toEmail := mail.NewEmail("", args.To)

	message := mail.NewSingleEmail(fromEmail, args.Subject, toEmail, args.TextBody, args.HTMLBody)

	client := sendgrid.NewSendClient(c.apiKey)
	response, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email via SendGrid: %w", err)
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("SendGrid returned error status %d: %s", response.StatusCode, response.Body)
	}

	return nil
}
