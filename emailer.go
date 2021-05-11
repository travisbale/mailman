package main

import (
	"fmt"
	"log"
	"os"

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

	from := mail.NewEmail("Tour Commissioner", "no-reply@manitobarydercup.com")
	subject := "Complete Registration for the Degenerate Golfers' Association Tour"
	to := mail.NewEmail(fmt.Sprintf("%s %s", invite.FirstName, invite.LastName), recipientAddress)
	plainContent := fmt.Sprintf("Click the following link to complete registration: %s", invite.Url)
	htmlContent := fmt.Sprintf("<strong>Click <a href='%s'>Here</a> to complete registration.</strong>", invite.Url)
	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)

	if err != nil {
		log.Println(err)
		return false
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
		return true
	}
}
