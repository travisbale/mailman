package sendgrid

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/travisbale/mailman/internal/email"
)

// Client implements EmailClient using SendGrid's API
type Client struct {
	apiKey string
}

// New creates a new SendGrid email client
func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

// Send sends an email via SendGrid
func (c *Client) Send(ctx context.Context, email email.Email) error {
	from := mail.NewEmail(email.FromName, email.From)
	to := mail.NewEmail("", email.To)

	message := mail.NewSingleEmail(from, email.Subject, to, email.TextBody, email.HTMLBody)

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
