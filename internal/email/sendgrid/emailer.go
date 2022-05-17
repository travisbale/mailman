package sendgrid

import (
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/travisbale/mailman/internal/email"
)

type emailer struct {
	client *sendgrid.Client
}

func NewService() email.Service {
	return &emailer{
		client: sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY")),
	}
}

func (e *emailer) Send(msg *email.Email) error {
	recipientAddress := os.Getenv("DEFAULT_RECIPIENT_ADDRESS")

	if os.Getenv("GOLANG_ENV") == "production" {
		recipientAddress = msg.Recipient.Address
	}

	sender := sgmail.NewEmail(msg.Sender.Name, msg.Sender.Address)
	recipient := sgmail.NewEmail(msg.Recipient.Name, recipientAddress)
	email := sgmail.NewSingleEmail(sender, msg.Subject, recipient, msg.PlainContent, msg.HtmlContent)

	response, err := e.client.Send(email)
	if err != nil {
		return fmt.Errorf("Send: %w", err)
	} else {
		log15.Debug(fmt.Sprintf("email sent to %s", recipientAddress), "status_code", response.StatusCode)
	}

	return nil
}
