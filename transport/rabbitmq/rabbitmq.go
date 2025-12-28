package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	ExchangeName = "points_exchange"
	RoutingKey   = "points.routing.key"
	QueueName    = "points_queue"
)

type RabbitMQ struct {
	logger *zap.Logger
	conn   *amqp.Connection
}

func NewRabbitMQ(logger *zap.Logger) *RabbitMQ {
	return &RabbitMQ{
		logger: logger,
	}
}

func (r *RabbitMQ) Connect(dsn string) error {
	// Initialize AMQP connection
	conn, err := amqp.Dial(dsn)
	if err != nil {
		r.logger.Error("Failed to connect to RabbitMQ", zap.Error(err))
		return err
	}
	r.conn = conn

	r.logger.Info("Successfully connected to RabbitMQ")
	return nil
}

func (r *RabbitMQ) DeclareExchange(exchange string) {
	if r.conn == nil {
		r.logger.Error("RabbitMQ connection not established")
		return
	}

	ch, err := r.conn.Channel()
	if err != nil {
		r.logger.Error("Failed to create channel for declaring exchange", zap.Error(err))
		return
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	)
	if err != nil {
		r.logger.Error("Failed to declare exchange", zap.String("exchange", exchange), zap.Error(err))
		return
	}

	r.logger.Info("Exchange declared", zap.String("exchange", exchange))
}

func (r *RabbitMQ) DeclareQueue(queue string) {
	if r.conn == nil {
		r.logger.Error("RabbitMQ connection not established")
		return
	}

	ch, err := r.conn.Channel()
	if err != nil {
		r.logger.Error("Failed to create channel for declaring queue", zap.Error(err))
		return
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		r.logger.Error("Failed to declare queue", zap.String("queue", queue), zap.Error(err))
		return
	}

	r.logger.Info("Queue declared", zap.String("queue", queue))
}

func (r *RabbitMQ) BindQueue(exchange, routingKey, queue string) {
	if r.conn == nil {
		r.logger.Error("RabbitMQ connection not established")
		return
	}

	ch, err := r.conn.Channel()
	if err != nil {
		r.logger.Error("Failed to create channel for binding queue", zap.Error(err))
		return
	}
	defer ch.Close()

	err = ch.QueueBind(
		queue,      // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,      // noWait
		nil,        // args
	)
	if err != nil {
		r.logger.Error("Failed to bind queue", zap.String("queue", queue), zap.String("exchange", exchange), zap.Error(err))
		return
	}

	r.logger.Info("Queue bound to exchange", zap.String("queue", queue), zap.String("exchange", exchange))
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, message []byte) error {
	if r.conn == nil {
		r.logger.Error("RabbitMQ connection not established")
		return fmt.Errorf("rabbitmq connection not established")
	}

	// Create a channel
	ch, err := r.conn.Channel()
	if err != nil {
		r.logger.Error("Failed to create channel", zap.Error(err))
		return err
	}
	defer ch.Close()

	// Declare the exchange
	err = ch.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // autoDelete
		false,    // internal
		false,    // noWait
		nil,      // args
	)
	if err != nil {
		r.logger.Error("Failed to declare exchange", zap.String("exchange", exchange), zap.Error(err))
		return err
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		"",    // queue name (empty means auto-generated)
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		r.logger.Error("Failed to declare queue", zap.String("exchange", exchange), zap.Error(err))
		return err
	}

	// Publish the message
	err = ch.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
	if err != nil {
		r.logger.Error("Failed to publish message", zap.String("exchange", exchange), zap.String("routingKey", routingKey), zap.Error(err))
		return err
	}

	r.logger.Info("Message published to exchange", zap.String("exchange", exchange), zap.String("routingKey", routingKey))
	return nil
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}

func (r *RabbitMQ) Read(ctx context.Context, queue string) ([]byte, error) {
	if r.conn == nil {
		r.logger.Error("RabbitMQ connection not established")
		return nil, fmt.Errorf("rabbitmq connection not established")
	}

	// Create a channel
	ch, err := r.conn.Channel()
	if err != nil {
		r.logger.Error("Failed to create channel", zap.Error(err))
		return nil, err
	}
	defer ch.Close()

	// Consume messages from the queue
	msgs, err := ch.Consume(
		queue, // queue
		"",    // consumer
		true,  // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		r.logger.Error("Failed to consume messages", zap.String("queue", queue), zap.Error(err))
		return nil, err
	}

	// Wait for a message
	select {
	case msg := <-msgs:
		r.logger.Info("Message received from queue", zap.String("queue", queue))
		return msg.Body, nil
	case <-ctx.Done():
		r.logger.Info("Context cancelled while waiting for message", zap.String("queue", queue))
		return nil, ctx.Err()
	}
}
