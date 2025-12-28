package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/k-tatiana/otus-project/internal/config"
	"github.com/k-tatiana/otus-project/internal/consumer"
	"github.com/k-tatiana/otus-project/internal/handlers/auth"
	"github.com/k-tatiana/otus-project/internal/handlers/points"
	"github.com/k-tatiana/otus-project/internal/handlers/users"
	"github.com/k-tatiana/otus-project/internal/metrics"
	"github.com/k-tatiana/otus-project/internal/middlewares"
	"github.com/k-tatiana/otus-project/internal/services"
	"github.com/k-tatiana/otus-project/internal/websocket"
	db "github.com/k-tatiana/otus-project/transport/postgres"
	"github.com/k-tatiana/otus-project/transport/rabbitmq"
	"github.com/k-tatiana/otus-project/transport/redis"
)

var ()

// RunServer starts the HTTP server on the provided port (if empty, default 8080).
func RunServer() {
	cfg := config.MustLoadConfig()
	ctx := context.Background()

	logger, _ := zap.NewDevelopment(
		zap.AddStacktrace(zap.ErrorLevel),
	)

	// Initialize metrics
	m := metrics.NewMetrics()

	// Initialize database connections
	pg, err := db.New(ctx, &cfg.MasterDatabaseURL, &cfg.SlaveDatabaseURL, cfg.EnableSlave, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database connections", zap.Error(err))
	}
	defer pg.Close(logger)

	redis, err := redis.New(ctx, cfg.RedisURL, logger)
	if err != nil {
		logger.Fatal("Failed to initialize redis connections", zap.Error(err))
	}
	defer redis.Close()

	sessionStore := services.NewSessionStore(redis)

	// Initialize RabbitMQ connection
	rmq := rabbitmq.NewRabbitMQ(logger)
	if err := rmq.Connect(cfg.RabbitMQURL); err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rmq.Close()

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	r := mux.NewRouter()

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply metrics middleware globally
	r.Use(metrics.HTTPMetricsMiddleware(m))

	// Mock OAuth-like flow endpoints.
	authHandler := auth.NewAuthHandler(sessionStore, logger, pg)
	r.HandleFunc("/auth/login", authHandler.LoginHandler).Methods(http.MethodGet)
	r.HandleFunc("/auth/callback", authHandler.CallbackHandler).Methods(http.MethodGet)
	r.HandleFunc("/auth/logout", authHandler.LogoutHandler).Methods(http.MethodGet)

	// WebSocket tunnel
	wsHandler := websocket.NewWebSocketHandler(sessionStore, logger)
	r.HandleFunc("/ws", wsHandler.ServeWS).Methods(http.MethodGet)

	// Protected API routes.
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middlewares.AuthMiddleware(sessionStore))
	pointsHandler := points.NewPointsHandler(sessionStore, pg, redis, wsHandler, logger)
	api.HandleFunc("/points/get", pointsHandler.GetPointsHandler).Methods(http.MethodGet)
	api.HandleFunc("/points/use", pointsHandler.UsePointsHandler).Methods(http.MethodPost)
	api.HandleFunc("/points/add", pointsHandler.AddPointsHandler).Methods(http.MethodPost)

	// User endpoints
	usersHandler := users.NewUsersHandler(pg, logger)
	api.HandleFunc("/users/get", usersHandler.GetUser).Methods(http.MethodGet)
	api.HandleFunc("/users/list", usersHandler.GetUsers).Methods(http.MethodGet)
	api.HandleFunc("/users/create", usersHandler.CreateUser).Methods(http.MethodPost)
	api.HandleFunc("/users/delete", usersHandler.DeleteUser).Methods(http.MethodDelete)

	// Start RabbitMQ consumer
	consumerHandler := consumer.NewConsumerHandler(logger, rmq, pg, wsHandler)
	consumerHandler.StartConsuming(ctx)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	logger.Info(fmt.Sprintf("Starting server on port: %s", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", zap.Error(err))
	}
}
