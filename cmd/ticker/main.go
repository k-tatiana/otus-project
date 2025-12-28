package main

import (
	"context"
	"log"

	external "github.com/k-tatiana/otus-project/external"
	"github.com/k-tatiana/otus-project/models"
	"github.com/k-tatiana/otus-project/transport/rabbitmq"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	cfg, err := external.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	sessionID := cfg.SessionID
	if sessionID == "" {
		sessionID = models.OrdersSessionID
	}

	logger := zap.NewNop()

	log.Printf("Starting orders cron job. Target: %s", baseURL)

	rmq := rabbitmq.NewRabbitMQ(logger)
	err = rmq.Connect(cfg.RMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	tickerService := external.NewTickerService(rmq, cfg.Intervals, true)
	tickerService.StartTickerCronJob(ctx, baseURL, sessionID)
}
