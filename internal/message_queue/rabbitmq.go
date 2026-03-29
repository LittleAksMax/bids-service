package message_queue

import (
	"encoding/json"
	"log"

	eval "github.com/LittleAksMax/bidscript/evaluator"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

func NewRabbitMQConnection(mqCfg *Config) MessageQueue {
	// https://medium.com/@george.benjamin.lopez/connecting-to-rabbitmq-using-golang-835b18d735a2
	conn, err := amqp.Dial(mqCfg.getConnectionURL())
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a RabbitMQ channel: %s", err)
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: ch,
	}
}

func (mq *RabbitMQClient) Publish(result *eval.Result) error {
	_, err := mq.channel.QueueDeclare(
		mq.queue, // e.g. "bids"
		true,     // durable
		false,    // auto-delete
		false,    // exclusive
		false,    // no-wait
		nil,
	)
	resultMap := map[string]interface{}{
		"operator":   string(result.Operator),
		"amount":     result.Amount,
		"percentage": result.Percentage,
	}
	body, err := json.Marshal(resultMap)
	if err != nil {
		return err
	}

	err = mq.channel.Publish(
		"",       // exchange
		mq.queue, // routing key (queue name)
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	return err
}

func (mq *RabbitMQClient) Close() error {
	return mq.conn.Close()
}
