package console

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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
	mu       sync.Mutex // Ensures atomic writes to stdout
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

	// Build entire output atomically to prevent interleaved output from concurrent workers
	var b strings.Builder
	b.WriteString("========================================\n")
	b.WriteString("📧 Email (Console Output)\n")
	b.WriteString("========================================\n")
	fmt.Fprintf(&b, "Sent: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "From: %s <%s>\n", email.FromName, email.From)
	fmt.Fprintf(&b, "To: %s\n", email.To)
	fmt.Fprintf(&b, "Subject: %s\n", rendered.Subject)
	b.WriteString("----------------------------------------\n")
	if rendered.TextBody != "" {
		b.WriteString("Text Body:\n")
		b.WriteString(rendered.TextBody)
		b.WriteString("\n")
		b.WriteString("----------------------------------------\n")
	}
	b.WriteString("========================================\n")

	// Lock to prevent interleaved writes from concurrent goroutines
	c.mu.Lock()
	fmt.Print(b.String())
	c.mu.Unlock()

	return nil
}
