package rabbitmq

import (
	"fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/streadway/amqp"
)

type MsgClient interface {
	SendMessage(message []byte) error
}

type MsgQueue struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func Open() *MsgQueue {
	user := os.Getenv("RABBITMQ_DEFAULT_USER")
	pass := os.Getenv("RABBITMQ_DEFAULT_PASS")
	host := os.Getenv("RABBITMQ_HOST")
	port := os.Getenv("RABBITMQ_PORT")

	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s", user, pass, host, port))
	if err != nil {
		panic(fmt.Sprintf("Listen: %s", err))
	}

	channel, err := conn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Listen: %s", err))
	}

	return &MsgQueue{
		connection: conn,
		channel:    channel,
	}
}

func (q *MsgQueue) RecieveMessages(queueName string, client MsgClient) {
	name := fmt.Sprintf("%s.%s", os.Getenv("RABBITMQ_PREFIX"), queueName)
	queue, err := q.channel.QueueDeclare(name, false, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("listen: %s", err))
	}

	msgs, err := q.channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		panic(fmt.Sprintf("listen: %s", err))
	}

	go func() {
		log15.Info("listening for messages", "queue", name)

		for msg := range msgs {
			if err := client.SendMessage(msg.Body); err != nil {
				log15.Error("failed to send message", "error", err)
			} else {
				log15.Info("message sent")
			}
		}
	}()
}

func (q *MsgQueue) Close() {
	q.connection.Close()
	q.channel.Close()
}
