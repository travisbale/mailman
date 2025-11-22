package sendgrid

import (
	"context"
	"fmt"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/travisbale/mailman/internal/email"
)

// renderer defines the interface for rendering email templates
type renderer interface {
	Render(ctx context.Context, templateName string, variables map[string]string) (*email.RenderedTemplate, error)
}

// Client implements EmailClient using SendGrid's API
type Client struct {
	apiKey   string
	renderer renderer
}

// New creates a new SendGrid email client
func New(apiKey string, renderer renderer) *Client {
	return &Client{
		apiKey:   apiKey,
		renderer: renderer,
	}
}

// Send renders the template and sends an email via SendGrid
func (c *Client) Send(ctx context.Context, email email.Email) error {
	rendered, err := c.renderer.Render(ctx, email.TemplateName, email.Variables)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	fromEmail := mail.NewEmail(email.FromName, email.From)
	toEmail := mail.NewEmail("", email.To)

	message := mail.NewSingleEmail(fromEmail, rendered.Subject, toEmail, rendered.TextBody, rendered.HTMLBody)

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
