package main

import (
	"os"

	"github.com/inconshreveable/log15"
	"github.com/travisbale/mailman/internal/email/sendgrid"
	"github.com/travisbale/mailman/internal/messages/invite"
	"github.com/travisbale/mailman/internal/messages/passwordreset"
	"github.com/travisbale/mailman/internal/rabbitmq"
	cli "github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "mailman",
		Usage: "sends html emails",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "username",
				Aliases: []string{"u"},
				Usage:   "RabbitMQ username",
				EnvVars:  []string{"RABBITMQ_DEFAULT_USER"},
			},
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"p"},
				Usage:   "RabbitMQ password",
				EnvVars:  []string{"RABBITMQ_DEFAULT_PASS"},
			},
			&cli.StringFlag{
				Name:   "hostname",
				Usage:  "Hostname of server running RabbitMQ",
				EnvVars: []string{"RABBITMQ_HOST"},
			},
			&cli.StringFlag{
				Name:   "port",
				Usage:  "Port number RabbitMQ is running on",
				EnvVars: []string{"RABBITMQ_PORT"},
			},
		},
		Action: func(c *cli.Context) error {
			emailClient := sendgrid.NewClient()
			inviteService := invite.NewService(emailClient)
			passwordService := passwordreset.NewService(emailClient)
			connection := rabbitmq.Open()

			forever := make(chan bool)

			connection.RecieveMessages("player_invitations", inviteService)
			connection.RecieveMessages("password_resets", passwordService)

			<-forever

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log15.Error("app failed to start", "error", err)
		return
	}
}
