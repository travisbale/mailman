package console

import (
	"context"
	"fmt"

	"github.com/travisbale/mailman/internal/email"
)

// renderer defines the interface for rendering email templates
type renderer interface {
	Render(ctx context.Context, templateName string, variables map[string]string) (*email.RenderedTemplate, error)
}

// Client implements EmailClient by printing emails to stdout
// This is useful for development and testing
type Client struct {
	renderer renderer
}

// New creates a new console email client
func New(renderer renderer) *Client {
	return &Client{
		renderer: renderer,
	}
}

// Send renders the template and prints the email to stdout instead of actually sending it
func (c *Client) Send(ctx context.Context, email email.Email) error {
	// Render the template
	rendered, err := c.renderer.Render(ctx, email.TemplateName, email.Variables)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Print to console
	fmt.Println("========================================")
	fmt.Println("📧 Email (Console Output)")
	fmt.Println("========================================")
	fmt.Printf("From: %s <%s>\n", email.FromName, email.From)
	fmt.Printf("To: %s\n", email.To)
	fmt.Printf("Subject: %s\n", rendered.Subject)
	fmt.Println("----------------------------------------")
	if rendered.TextBody != "" {
		fmt.Println("Text Body:")
		fmt.Println(rendered.TextBody)
		fmt.Println("----------------------------------------")
	}
	fmt.Println("========================================")
	return nil
}
