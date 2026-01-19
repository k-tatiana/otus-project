package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	dsn    string
	conn   *amqp.Connection
	chPool chan *amqp.Channel
	mux    sync.RWMutex
	closed bool
}

const (
	channelPoolSize = 10
	reconnectDelay  = 5 * time.Second
)

func NewRabbitMQ(logger *zap.Logger) *RabbitMQ {
	return &RabbitMQ{
		logger: logger,
		chPool: make(chan *amqp.Channel, channelPoolSize),
		closed: false,
	}
}

func (r *RabbitMQ) Connect(dsn string) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.dsn = dsn
	if err := r.connect(); err != nil {
		return err
	}

	// Initialize channel pool
	for i := 0; i < channelPoolSize; i++ {
		ch, err := r.conn.Channel()
		if err != nil {
			r.logger.Error("Failed to create channel for pool", zap.Error(err))
			continue
		}
		r.chPool <- ch
	}

	// Start connection monitor
	go r.monitorConnection()

	r.logger.Info("Successfully connected to RabbitMQ")
	return nil
}

func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(r.dsn)
	if err != nil {
		r.logger.Error("Failed to connect to RabbitMQ", zap.Error(err))
		return err
	}
	r.conn = conn
	return nil
}

func (r *RabbitMQ) monitorConnection() {
	for {
		r.mux.RLock()
		if r.closed {
			r.mux.RUnlock()
			return
		}
		r.mux.RUnlock()

		reason, ok := <-r.conn.NotifyClose(make(chan *amqp.Error))
		if !ok {
			r.logger.Info("Connection closed", zap.Error(reason))
			return
		}

		r.logger.Error("Connection closed", zap.Error(reason))

		for {
			r.mux.Lock()
			if err := r.connect(); err == nil {
				r.mux.Unlock()
				break
			}
			r.mux.Unlock()

			r.logger.Info("Failed to reconnect. Retrying...", zap.Duration("delay", reconnectDelay))
			time.Sleep(reconnectDelay)
		}

		// Reinitialize channel pool after reconnection
		r.reinitializeChannelPool()
	}
}

func (r *RabbitMQ) reinitializeChannelPool() {
	// Clear existing pool
	for len(r.chPool) > 0 {
		ch := <-r.chPool
		ch.Close()
	}

	// Refill pool
	for i := 0; i < channelPoolSize; i++ {
		ch, err := r.conn.Channel()
		if err != nil {
			r.logger.Error("Failed to create channel for pool during reinitialization", zap.Error(err))
			continue
		}
		r.chPool <- ch
	}
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

func (r *RabbitMQ) getChannel() (*amqp.Channel, error) {
	r.mux.RLock()
	if r.conn == nil || r.closed {
		r.mux.RUnlock()
		return nil, fmt.Errorf("connection not established or closed")
	}
	r.mux.RUnlock()

	// Try to get a channel from the pool
	select {
	case ch := <-r.chPool:
		// Verify channel is still open
		if ch.IsClosed() {
			// Create new channel if this one is closed
			newCh, err := r.conn.Channel()
			if err != nil {
				return nil, err
			}
			return newCh, nil
		}
		return ch, nil
	default:
		// If pool is empty, create a new channel
		return r.conn.Channel()
	}
}

func (r *RabbitMQ) returnChannel(ch *amqp.Channel) {
	if ch != nil && !ch.IsClosed() {
		// Try to return to pool, if full, close the channel
		select {
		case r.chPool <- ch:
		default:
			ch.Close()
		}
	}
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, message []byte) error {
	ch, err := r.getChannel()
	if err != nil {
		r.logger.Error("Failed to get channel", zap.Error(err))
		return err
	}
	defer r.returnChannel(ch)

	// Publish the message
	err = ch.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         message,
			DeliveryMode: amqp.Persistent,
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
	r.mux.Lock()
	r.closed = true
	r.mux.Unlock()

	// Close all channels in the pool
	for len(r.chPool) > 0 {
		ch := <-r.chPool
		ch.Close()
	}

	// Close the connection
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
