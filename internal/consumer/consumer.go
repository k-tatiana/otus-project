package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/k-tatiana/otus-project/internal/websocket"
	"github.com/k-tatiana/otus-project/models"
	"github.com/k-tatiana/otus-project/transport/postgres"
	"github.com/k-tatiana/otus-project/transport/rabbitmq"
)

type ConsumerHandler struct {
	logger *zap.Logger
	rmq    *rabbitmq.RabbitMQ
	db     *postgres.DB
	ws     *websocket.WebSocketHandler
}

func NewConsumerHandler(logger *zap.Logger, rmq *rabbitmq.RabbitMQ, db *postgres.DB, ws *websocket.WebSocketHandler) *ConsumerHandler {
	return &ConsumerHandler{
		logger: logger,
		rmq:    rmq,
		db:     db,
		ws:     ws,
	}
}

func (c *ConsumerHandler) StartConsuming(ctx context.Context) {
	c.logger.Info("Starting RabbitMQ consumer")

	// Ensure exchange and queue are declared
	c.rmq.DeclareExchange(rabbitmq.ExchangeName)
	c.rmq.DeclareQueue(rabbitmq.QueueName)
	c.rmq.BindQueue(rabbitmq.ExchangeName, rabbitmq.RoutingKey, rabbitmq.QueueName)

	go c.consumeMessages(ctx)
}

func (c *ConsumerHandler) consumeMessages(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer context cancelled")
			return
		case <-ticker.C:
			c.processMessage(ctx)
		}
	}
}

func (c *ConsumerHandler) processMessage(ctx context.Context) {
	message, err := c.rmq.Read(ctx, rabbitmq.QueueName)
	if err != nil {
		if err == context.DeadlineExceeded {
			return
		}
		c.logger.Error("Failed to read message from queue", zap.Error(err))
		return
	}

	if message == nil {
		return
	}

	var msg models.Message
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Error("Failed to unmarshal message", zap.Error(err), zap.String("message", string(message)))
		return
	}

	c.logger.Info("Processing message from queue",
		zap.String("customer_id", msg.CustomerID),
		zap.Int("amount", msg.Amount),
		zap.Int("reason_id", msg.ReasonID))

	if err := c.addPointsToDatabase(ctx, msg); err != nil {
		c.logger.Error("Failed to add points to database", zap.Error(err))
		return
	}

	reasonName := c.getReasonName(msg.ReasonID)

	c.broadcastBalanceUpdate(msg.CustomerID, msg.Amount, reasonName)

	c.logger.Info("Successfully processed message",
		zap.String("customer_id", msg.CustomerID),
		zap.Int("amount", msg.Amount),
		zap.Int("reason_id", msg.ReasonID))
}

func (c *ConsumerHandler) getReasonName(d int) string {
	row := c.db.Slave.QueryRow(context.Background(), "SELECT name FROM balance_reasons WHERE id = $1", d)
	var name string
	if err := row.Scan(&name); err != nil {
		c.logger.Error("Failed to get reason name", zap.Error(err))
		return ""
	}
	return name
}

func (c *ConsumerHandler) addPointsToDatabase(ctx context.Context, msg models.Message) error {
	tx, err := c.db.Master.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if customer exists
	var customerID int
	if err = tx.QueryRow(ctx, "SELECT id FROM customers WHERE id = $1", msg.CustomerID).Scan(&customerID); err != nil {
		return fmt.Errorf("customer not found: %w", err)
	}

	// Get current balance
	var currentBalance int
	if err = tx.QueryRow(ctx, "SELECT balance FROM loyalty WHERE customer_id = $1 FOR UPDATE", msg.CustomerID).Scan(&currentBalance); err != nil {
		// If no loyalty record exists, create one
		if _, err = tx.Exec(ctx, "INSERT INTO loyalty (customer_id, balance, level_id) VALUES ($1, $2, $3)", msg.CustomerID, 0, 1); err != nil {
			return fmt.Errorf("failed to create loyalty record: %w", err)
		}
		currentBalance = 0
	}

	// Update balance
	if _, err = tx.Exec(ctx, "UPDATE loyalty SET balance = balance + $1 WHERE customer_id = $2", msg.Amount, msg.CustomerID); err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Insert balance history
	if _, err = tx.Exec(ctx, "INSERT INTO balance_history (customer_id, points, created_at, reason_id) VALUES ($1, $2, $3, $4)", msg.CustomerID, msg.Amount, time.Now(), msg.ReasonID); err != nil {
		return fmt.Errorf("failed to insert balance history: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (c *ConsumerHandler) broadcastBalanceUpdate(userID string, balance int, reasonName string) {
	messageBytes, err := json.Marshal(map[string]interface{}{
		"reason": reasonName,
		"data": map[string]interface{}{
			"user_id":     userID,
			"points_diff": balance,
		},
	})
	if err != nil {
		c.logger.Error("failed to marshal message", zap.Error(err))
		return
	}
	// sending message into websocket
	c.ws.Broadcast(messageBytes)
}
