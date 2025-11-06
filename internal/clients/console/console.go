package console

import (
	"context"
	"fmt"

	"github.com/travisbale/mailman/internal/email"
)

// Client implements EmailClient by printing emails to stdout
// This is useful for development and testing
type Client struct{}

// New creates a new console email client
func New() *Client {
	return &Client{}
}

// Send prints the email to stdout instead of actually sending it
func (c *Client) Send(ctx context.Context, email email.Email) error {
	fmt.Println("========================================")
	fmt.Println("📧 Email (Console Output)")
	fmt.Println("========================================")
	fmt.Printf("From: %s <%s>\n", email.FromName, email.From)
	fmt.Printf("To: %s\n", email.To)
	fmt.Printf("Subject: %s\n", email.Subject)
	fmt.Println("----------------------------------------")
	if email.TextBody != "" {
		fmt.Println("Text Body:")
		fmt.Println(email.TextBody)
		fmt.Println("----------------------------------------")
	}
	if email.HTMLBody != "" {
		fmt.Println("HTML Body:")
		fmt.Println(email.HTMLBody)
	}
	fmt.Println("========================================")
	return nil
}
