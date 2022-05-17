package passwordreset

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"

	"github.com/travisbale/mailman/internal/email"
)

type MessageData struct {
	Email string
	Url   string
}

type resetService struct {
	tmplate      *template.Template
	emailService email.Service
}

type Service interface {
	SendPasswordReset(details MessageData) error
}

func NewService(emailService email.Service) Service {
	files := []string{"templates/base.html", "templates/reset-password.html"}
	tmplate, err := template.ParseFiles(files...)
	if err != nil {
		panic(fmt.Sprintf("NewService: %s", err))
	}

	return &resetService{
		tmplate:      tmplate,
		emailService: emailService,
	}
}

func (s *resetService) SendPasswordReset(data MessageData) error {
	htmlContent := new(bytes.Buffer)
	if err := s.tmplate.ExecuteTemplate(htmlContent, "base", data); err != nil {
		return fmt.Errorf("SendPasswordReset: %w", err)
	}

	email := &email.Email{
		Subject:      "Manitoba Ryder Cup password reset request",
		Sender:       &mail.Address{Name: "Manitoba Ryder Cup", Address: "no-reply@manitobarydercup.com"},
		Recipient:    &mail.Address{Name: "", Address: data.Email},
		PlainContent: fmt.Sprintf("Click the following link to reset your password: %s", data.Url),
		HtmlContent:  htmlContent.String(),
	}

	if err := s.emailService.Send(email); err != nil {
		return fmt.Errorf("SendPasswordReset: %w", err)
	}

	return nil
}
