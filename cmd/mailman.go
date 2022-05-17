package main

import (
	"os"

	"github.com/inconshreveable/log15"
	"github.com/travisbale/mailman/internal/email/sendgrid"
	"github.com/travisbale/mailman/internal/messages/invite"
	"github.com/travisbale/mailman/internal/messages/passwordreset"
	"github.com/travisbale/mailman/internal/rabbitmq"
	"github.com/urfave/cli"
)

func main() {
	app := &cli.App{
		Name:  "mailman",
		Usage: "sends html emails",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "username, u",
				Usage:  "RabbitMQ username",
				EnvVar: "RABBITMQ_DEFAULT_USER",
			},
			cli.StringFlag{
				Name:   "password, p",
				Usage:  "RabbitMQ password",
				EnvVar: "RABBITMQ_DEFAULT_PASS",
			},
			cli.StringFlag{
				Name:   "hostname",
				Usage:  "Hostname of server running RabbitMQ",
				EnvVar: "RABBITMQ_HOST",
			},
			cli.StringFlag{
				Name:   "port",
				Usage:  "Port number RabbitMQ is running on",
				EnvVar: "RABBITMQ_PORT",
			},
		},
		Action: func(c *cli.Context) error {
			emailService := sendgrid.NewService()
			inviteService := invite.NewService(emailService)
			passwordResetService := passwordreset.NewService(emailService)

			rabbitmq.Listen(inviteService, passwordResetService)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log15.Error("app failed to start", "error", err)
		return
	}
}
