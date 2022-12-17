package passwordreset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/mail"

	"github.com/travisbale/mailman/internal/email"
)

type message struct {
	Email string
	Url   string
}

type service struct {
	tmplate     *template.Template
	emailClient EmailClient
}

type EmailClient interface {
	SendEmail(email *email.Email) error
}

func NewService(emailClient EmailClient) *service {
	files := []string{"templates/base.html", "templates/reset-password.html"}
	tmplate, err := template.ParseFiles(files...)
	if err != nil {
		panic(fmt.Sprintf("NewService: %s", err))
	}

	return &service{
		tmplate:     tmplate,
		emailClient: emailClient,
	}
}

func (s *service) SendMessage(data []byte) error {
	var msg message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("SendMessage: %w", err)
	}

	htmlContent := new(bytes.Buffer)
	if err := s.tmplate.ExecuteTemplate(htmlContent, "base", msg); err != nil {
		return fmt.Errorf("SendMessage: %w", err)
	}

	email := &email.Email{
		Subject:      "Ryder Cup password reset request",
		Sender:       &mail.Address{Name: "Ryder Cup Support", Address: "no-reply@manitobarydercup.com"},
		Recipient:    &mail.Address{Name: "", Address: msg.Email},
		PlainContent: fmt.Sprintf("Click the following link to reset your password: %s", msg.Url),
		HtmlContent:  htmlContent.String(),
	}

	if err := s.emailClient.SendEmail(email); err != nil {
		return fmt.Errorf("SendMessage: %w", err)
	}

	return nil
}
