package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"html/template"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type PlayerInvitation struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Url       string `json:"url"`
}

func (invite *PlayerInvitation) Send() bool {
	var recipientAddress string

	// Be careful not to send email to actual recipients out of production
	if os.Getenv("GOLANG_ENV") == "production" {
		recipientAddress = invite.Email
	} else {
		recipientAddress = os.Getenv("DEFAULT_RECIPIENT_ADDRESS")
	}

	t, err := template.ParseFiles("templates/player-invitation.html")
	if err != nil {
		failOnError(err, "Failed to parse template")
	}

	htmlContent := new(bytes.Buffer)
	if err = t.Execute(htmlContent, invite); err != nil {
		failOnError(err, "Failed to execute template")
	}

	from := mail.NewEmail("DGA Tour Commissioner", "no-reply@manitobarydercup.com")
	subject := "The Manitoba Ryder Cup"
	to := mail.NewEmail(fmt.Sprintf("%s %s", invite.FirstName, invite.LastName), recipientAddress)
	plainContent := fmt.Sprintf("Click the following link to complete registration: %s", invite.Url)
	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent.String())
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)

	if err != nil {
		log.Printf("%s", err)
		return false
	} else {
		log.Printf("%d", response.StatusCode)
		log.Printf("%s", response.Body)
		log.Printf("%s", response.Headers)
		return true
	}
}
