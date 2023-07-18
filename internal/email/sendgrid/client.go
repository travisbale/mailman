package sendgrid

import (
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/travisbale/mailman/internal/email"
)

type SendGridClient struct {
	client *sendgrid.Client
}

func NewClient() *SendGridClient {
	return &SendGridClient{
		client: sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY")),
	}
}

func (e *SendGridClient) SendEmail(msg *email.Email) error {
	recipientAddress := os.Getenv("DEFAULT_RECIPIENT_ADDRESS")

	if os.Getenv("GOLANG_ENV") == "production" {
		recipientAddress = msg.Recipient.Address
	}

	sender := mail.NewEmail(msg.Sender.Name, msg.Sender.Address)
	recipient := mail.NewEmail(msg.Recipient.Name, recipientAddress)
	email := mail.NewSingleEmail(sender, msg.Subject, recipient, msg.PlainContent, msg.HtmlContent)

	response, err := e.client.Send(email)
	if err != nil {
		return fmt.Errorf("Send: %w", err)
	}

	log15.Debug(fmt.Sprintf("email sent to %s", recipientAddress), "status_code", response.StatusCode)

	return nil
}
