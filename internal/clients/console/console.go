package console

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/travisbale/mailman/internal/email"
)

// Client implements email delivery by printing emails to stdout
type Client struct {
	mu sync.Mutex // Prevents interleaved output from concurrent workers
}

// New creates a new console email client
func New() *Client {
	return &Client{}
}

// Send prints a pre-rendered email to stdout
func (c *Client) Send(ctx context.Context, args email.JobArgs) error {
	var b strings.Builder
	b.WriteString("========================================\n")
	b.WriteString("📧 Email (Console Output)\n")
	b.WriteString("========================================\n")
	fmt.Fprintf(&b, "Sent: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "From: %s <%s>\n", args.FromName, args.From)
	fmt.Fprintf(&b, "To: %s\n", args.To)
	fmt.Fprintf(&b, "Subject: %s\n", args.Subject)
	b.WriteString("----------------------------------------\n")
	if args.HTMLBody != "" {
		b.WriteString("HTML Body:\n")
		b.WriteString(args.HTMLBody)
		b.WriteString("\n")
	}
	if args.TextBody != "" {
		b.WriteString("Text Body:\n")
		b.WriteString(args.TextBody)
		b.WriteString("\n")
	}
	b.WriteString("========================================\n")

	c.mu.Lock()
	fmt.Print(b.String())
	c.mu.Unlock()

	return nil
}
