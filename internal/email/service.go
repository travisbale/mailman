package email

import "net/mail"

type Email struct {
	Subject      string
	Sender       *mail.Address
	Recipient    *mail.Address
	PlainContent string
	HtmlContent  string
}

type Service interface {
	Send(msg *Email) error
}
