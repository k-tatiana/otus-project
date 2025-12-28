package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/k-tatiana/otus-project/models"
	"github.com/k-tatiana/otus-project/transport/rabbitmq"
)

type TickerService struct {
	rmq          *rabbitmq.RabbitMQ
	intervals    Intervals
	rmqAvailable bool
}

func NewTickerService(rmq *rabbitmq.RabbitMQ, intervals Intervals, isQueueAvailable bool) *TickerService {
	return &TickerService{
		rmq:          rmq,
		intervals:    intervals,
		rmqAvailable: isQueueAvailable,
	}
}

func (ts *TickerService) StartTickerCronJob(ctx context.Context, baseURL, sessionID string) {
	time.Sleep(2 * time.Second)
	var (
		handlerAddPoints    = baseURL + string(models.HandlerPointsAdd)
		handlerDeletePoints = baseURL + string(models.HandlerPointsUse)
	)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	userIDs := ts.getUserIDs(client, baseURL, sessionID)
	if len(userIDs) == 0 {
		log.Println("No users found, skipping ticker job")
		return
	}
	log.Printf("Found %d users, starting ticker job", len(userIDs))

	ordersTicker := time.NewTicker(ts.intervals.OrdersCron)
	defer ordersTicker.Stop()
	log.Printf("Orders ticker started with interval: %v", ts.intervals.OrdersCron)

	marketingTicker := time.NewTicker(ts.intervals.MarketingCron)
	defer marketingTicker.Stop()
	log.Printf("Marketing ticker started with interval: %v", ts.intervals.MarketingCron)

	usingPointsTicker := time.NewTicker(ts.intervals.UsingPointsCron)
	defer usingPointsTicker.Stop()
	log.Printf("Using points ticker started with interval: %v", ts.intervals.UsingPointsCron)

	if !ts.rmqAvailable {
		log.Println("RabbitMQ is not available, skipping RabbitMQ publisher")
	} else {
		prepareRmq := func() {
			// Ensure RabbitMQ exchange and queue are declared
			ts.rmq.DeclareExchange(rabbitmq.ExchangeName)
			ts.rmq.DeclareQueue(rabbitmq.QueueName)
			ts.rmq.BindQueue(rabbitmq.ExchangeName, rabbitmq.RoutingKey, rabbitmq.QueueName)
		}
		prepareRmq()
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Orders cron job cancelled")
			return
		case <-ordersTicker.C:
			randUserID := userIDs[rand.Intn(len(userIDs))]
			randPoints := rand.Intn(100) + 1
			if ts.rmqAvailable {
				ts.startRabbitMQPublisher(ctx, randUserID, models.ReasonOrder, randPoints)
			} else {
				ts.runAddToDatabase(client, randUserID, handlerAddPoints, sessionID, models.ReasonOrder)
			}
			fmt.Println("Orders ticker tick")
		case <-marketingTicker.C:
			randUserID := userIDs[rand.Intn(len(userIDs))]
			randPoints := rand.Intn(100) + 1
			if ts.rmqAvailable {
				ts.startRabbitMQPublisher(ctx, randUserID, models.ReasonMarketingActivity, randPoints)
			} else {
				ts.runAddToDatabase(client, randUserID, handlerAddPoints, sessionID, models.ReasonMarketingActivity)
			}
			ts.runAddToDatabase(client, randUserID, handlerAddPoints, sessionID, models.ReasonMarketingActivity)
			fmt.Println("Marketing ticker tick")
		case <-usingPointsTicker.C:
			randUserID := userIDs[rand.Intn(len(userIDs))]
			randPoints := (rand.Intn(100) + 1) * -1
			if ts.rmqAvailable {
				ts.startRabbitMQPublisher(ctx, randUserID, models.ReasonUsingPoints, randPoints)
			} else {
				ts.runAddToDatabase(client, randUserID, handlerDeletePoints, sessionID, models.ReasonUsingPoints)
			}
			fmt.Println("Using points ticker tick")

		}
	}

}

// startRabbitMQPublisher
func (ts *TickerService) startRabbitMQPublisher(ctx context.Context, randUserID int, reason int, points int) {

	msg := models.Message{
		CustomerID: strconv.Itoa(randUserID),
		Amount:     points,
		ReasonID:   reason,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Cron: failed to marshal message: %v", err)
		return
	}

	ts.rmq.Publish(ctx, rabbitmq.ExchangeName, rabbitmq.RoutingKey, payload)
	log.Printf("Cron: Published message for user %d with %d points", randUserID, points)
}

// runAddToDatabase adding points into database by requesting handler - without queue
func (ts *TickerService) runAddToDatabase(client *http.Client, randUserID int, baseURL, sessionID string, reason_id int) {
	randPoints := rand.Intn(100)
	targetURL := baseURL + "?user_id=" + strconv.Itoa(randUserID)
	log.Printf("Cron: target URL: %s", targetURL)

	msg := models.Message{
		Amount:     randPoints,
		CustomerID: strconv.Itoa(randUserID),
		ReasonID:   reason_id,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Cron: failed to marshal payload: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Cron: failed to create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Id", sessionID)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Cron: request failed: %v", err)
		return
	}

	// Read and discard body to reuse connection
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Cron: request failed with status: %s", resp.Status)
	} else {
		log.Printf("Cron: successfully added points")
	}
}

// getUserIDs retrieves all customer IDs from the database
func (ts *TickerService) getUserIDs(client *http.Client, baseURL, sessionID string) []int {
	var (
		users   []models.User
		userIDs []int
	)

	targetURL := baseURL + string(models.HandlerUsersList)
	log.Printf("Get users url: %s", targetURL)

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		log.Fatalf("Failed to request user list: %v", err)
		return userIDs
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Id", sessionID)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Users request failed: %v", err)
		return userIDs
	}

	// Read and discard body to reuse connection
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return userIDs
	}
	log.Printf("Response body: %s", string(respBody))

	// Parse the response to extract user IDs
	// Assuming response is a JSON array of user objects with ID field
	// You may need to adjust this parsing based on actual API response format
	if err := json.Unmarshal(respBody, &users); err != nil {
		log.Fatalf("Failed to parse response body: %v", err)
		return userIDs
	}

	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	log.Printf("Found %d users in database", len(userIDs))
	return userIDs
}
