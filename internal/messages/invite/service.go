package invite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/mail"

	"github.com/inconshreveable/log15"
	"github.com/travisbale/mailman/internal/email"
)

type message struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Url       string `json:"url"`
}

type service struct {
	template    *template.Template
	emailClient EmailClient
}

type EmailClient interface {
	SendEmail(email *email.Email) error
}

func NewService(emailClient EmailClient) *service {
	files := []string{"templates/base.html", "templates/player-invitation.html"}
	template, err := template.ParseFiles(files...)
	if err != nil {
		panic(fmt.Sprintf("NewService: %s", err))
	}

	return &service{
		template:    template,
		emailClient: emailClient,
	}
}

func (s *service) SendMessage(data []byte) error {
	var message message
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("SendMessage: %w", err)
	}

	htmlContent := new(bytes.Buffer)
	if err := s.template.ExecuteTemplate(htmlContent, "base", message); err != nil {
		log15.Debug(s.template.DefinedTemplates())
		return fmt.Errorf("SendMessage: %w", err)
	}

	email := &email.Email{
		Subject:      "Welcome to the Manitoba Ryder Cup!",
		Sender:       &mail.Address{Name: "Ryder Cup Commissioner", Address: "no-reply@manitobarydercup.com"},
		Recipient:    &mail.Address{Name: fmt.Sprintf("%s %s", message.FirstName, message.LastName), Address: message.Email},
		PlainContent: fmt.Sprintf("Click the following link to complete registration: %s", message.Url),
		HtmlContent:  htmlContent.String(),
	}

	if err := s.emailClient.SendEmail(email); err != nil {
		return fmt.Errorf("SendMessage: %w", err)
	}

	return nil
}
