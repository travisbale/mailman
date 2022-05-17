package rabbitmq

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/streadway/amqp"
	"github.com/travisbale/mailman/internal/messages/invite"
	"github.com/travisbale/mailman/internal/messages/passwordreset"
)

func Listen(inviteService invite.Service, passwordResetService passwordreset.Service) {
	user := os.Getenv("RABBITMQ_DEFAULT_USER")
	pass := os.Getenv("RABBITMQ_DEFAULT_PASS")
	host := os.Getenv("RABBITMQ_HOST")
	port := os.Getenv("RABBITMQ_PORT")

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s", user, pass, host, port))
	if err != nil {
		panic(fmt.Sprintf("Listen: %s", err))
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Listen: %s", err))
	}
	defer channel.Close()

	forever := make(chan bool)

	listenForInvites(inviteService, channel)
	listenForPasswordResetRequests(passwordResetService, channel)

	<-forever
}

func listenForInvites(inviteService invite.Service, channel *amqp.Channel) {
	queueName := fmt.Sprintf("%s.player_invitations", os.Getenv("RABBITMQ_PREFIX"))
	queue, err := channel.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("listenForInvites: %s", err))
	}

	messages, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("listenForInvites: %s", err))
	}

	go func() {
		log15.Info("listening for player invitations")

		for message := range messages {
			var data invite.MessageData

			if err := json.Unmarshal(message.Body, &data); err != nil {
				log15.Error("failed to parse player invitation", "error", err)
			} else {
				log15.Info("player invitation received", "data", data)

				if err := inviteService.SendInvitation(data); err != nil {
					log15.Error("failed to send email", "error", err)
				} else {
					log15.Info("player invitation sent")
				}
			}

		}
	}()
}

func listenForPasswordResetRequests(passwordResetService passwordreset.Service, channel *amqp.Channel) {
	queueName := fmt.Sprintf("%s.password_resets", os.Getenv("RABBITMQ_PREFIX"))
	queue, err := channel.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("listenForPasswordResetRequests: %s", err))
	}

	messages, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("listenForPasswordResetRequests: %s", err))
	}

	go func() {
		log15.Info("listening for password resets")

		for message := range messages {
			var data passwordreset.MessageData

			if err := json.Unmarshal(message.Body, &data); err != nil {
				log15.Error("failed to parse password reset message", "error", err)
			} else {
				log15.Info("password reset request received", "data", data)

				if err := passwordResetService.SendPasswordReset(data); err != nil {
					log15.Error("failed to sernd email", "error", err)
				} else {
					log15.Info("password reset request sent")
				}
			}

		}
	}()
}
