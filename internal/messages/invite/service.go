package invite

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"

	"github.com/inconshreveable/log15"
	"github.com/travisbale/mailman/internal/email"
)

type MessageData struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Url       string `json:"url"`
}

type inviteService struct {
	template     *template.Template
	emailService email.Service
}

type Service interface {
	SendInvitation(details MessageData) error
}

func NewService(emailService email.Service) Service {
	files := []string{"templates/base.html", "templates/player-invitation.html"}
	template, err := template.ParseFiles(files...)
	if err != nil {
		panic(fmt.Sprintf("NewService: %s", err))
	}

	return &inviteService{
		template:     template,
		emailService: emailService,
	}
}

func (s *inviteService) SendInvitation(data MessageData) error {
	htmlContent := new(bytes.Buffer)
	if err := s.template.ExecuteTemplate(htmlContent, "base", data); err != nil {
		log15.Debug(s.template.DefinedTemplates())
		return fmt.Errorf("SendInvitation: %w", err)
	}

	email := &email.Email{
		Subject:      "Welcome to the Manitoba Ryder Cup!",
		Sender:       &mail.Address{Name: "Tournament Commissioner", Address: "no-reply@manitobarydercup.com"},
		Recipient:    &mail.Address{Name: fmt.Sprintf("%s %s", data.FirstName, data.LastName), Address: data.Email},
		PlainContent: fmt.Sprintf("Click the following link to complete registration: %s", data.Url),
		HtmlContent:  htmlContent.String(),
	}

	if err := s.emailService.Send(email); err != nil {
		return fmt.Errorf("SendInvitation: %w", err)
	}

	return nil
}
