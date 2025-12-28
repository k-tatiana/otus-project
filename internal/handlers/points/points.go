package points

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	"github.com/k-tatiana/otus-project/internal/services"
	"github.com/k-tatiana/otus-project/internal/websocket"
	"github.com/k-tatiana/otus-project/models"
	postgres "github.com/k-tatiana/otus-project/transport/postgres"
	redis "github.com/k-tatiana/otus-project/transport/redis"
)

const pointsRedisTTL = 24 * time.Hour

// PointsHandler handles loyalty points related requests.
type PointsHandler struct {
	sessionStore *services.SessionStore
	pg           *postgres.DB
	redis        *redis.Client
	ws           *websocket.WebSocketHandler
	logger       *zap.Logger
}

// NewPointsHandler creates a new PointsHandler instance.
func NewPointsHandler(
	sessionStore *services.SessionStore,
	pg *postgres.DB,
	redis *redis.Client,
	ws *websocket.WebSocketHandler,
	logger *zap.Logger,
) *PointsHandler {
	return &PointsHandler{
		pg:           pg,
		redis:        redis,
		ws:           ws,
		sessionStore: sessionStore,
		logger:       logger,
	}
}

// GetPointsHandler returns loyalty points information for the current user.
func (h *PointsHandler) GetPointsHandler(w http.ResponseWriter, r *http.Request) {
	var balance int
	var level string

	ctx := r.Context()

	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		h.logger.Error("user_id is required")
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	var redisKey = "points:" + userID
	existsInCache, err := h.redis.Exists(ctx, redisKey)
	if err != nil {
		h.logger.Error("failed to check points in cache", zap.Error(err))
		http.Error(w, "failed to check points in cache", http.StatusInternalServerError)
		return
	}
	if existsInCache > 0 {
		balanceLevel, err := h.redis.Get(ctx, redisKey)
		if err != nil {
			h.logger.Error("failed to get points from cache", zap.Error(err))
			http.Error(w, "failed to get points from cache", http.StatusInternalServerError)
			return
		}
		balanceLevelArr := strings.Split(balanceLevel, "_")
		balance, _ = strconv.Atoi(balanceLevelArr[0])
		level = string(balanceLevelArr[1])
	} else {
		// Query from slave (read replica) for read operations
		err := h.pg.Slave.QueryRow(ctx,
			`SELECT loyalty.balance, loyalty_levels.name as level_name
		FROM loyalty 
		LEFT JOIN loyalty_levels ON loyalty.level_id = loyalty_levels.id 
		WHERE loyalty.customer_id = $1`, userID,
		).Scan(&balance, &level)
		if err != nil && err != pgx.ErrNoRows {
			h.logger.Error("failed to fetch points", zap.Error(err))
			http.Error(w, "failed to fetch points", http.StatusInternalServerError)
			return
		}

		if err = h.redis.Set(ctx, redisKey, fmt.Sprintf("%d_%s", balance, level), pointsRedisTTL); err != nil {
			h.logger.Error("failed to set up cache for user", zap.String("user_id", userID), zap.Error(err))
			http.Error(w, "failed to set up cache", http.StatusInternalServerError)
			return
		}
	}

	// Mock response for now
	w.Header().Set("Content-Type", "application/json")
	resp := models.PointsInfo{
		UserID:  userID,
		Balance: balance,
		Level:   level,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UsePointsHandler handles the deduction of points from a user's account.
func (h *PointsHandler) UsePointsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.logger.Error("user_id is required")
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Parse request body
	var req struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request body", zap.Error(err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		h.logger.Error("balance using amount must be positive")
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := h.pg.Master.Begin(ctx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Check current balance
	info := models.PointsInfo{UserID: userID}
	err = tx.QueryRow(ctx,
		`SELECT l.balance, ll.name 
		FROM loyalty l 
		JOIN loyalty_levels ll on l.level_id = ll.id 
		WHERE customer_id = $1 FOR UPDATE`, userID).Scan(&info.Balance, &info.Level)
	if err != nil {
		h.logger.Error("failed to fetch points", zap.Error(err))
		http.Error(w, "user balance is empty", http.StatusNotFound)
		return
	}

	if info.Balance < req.Amount {
		h.logger.Error("insufficient funds")
		http.Error(w, "insufficient funds", http.StatusPaymentRequired)
		return
	}

	// Deduct points
	_, err = tx.Exec(ctx, "UPDATE loyalty SET balance = balance - $1 WHERE customer_id = $2", req.Amount, userID)
	if err != nil {
		h.logger.Error("failed to update points", zap.Error(err))
		http.Error(w, "failed to update balance", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(ctx, "INSERT INTO balance_history (customer_id, points, created_at) VALUES ($1, $2, $3)", userID, -req.Amount, time.Now())
	if err != nil {
		h.logger.Error("failed to insert balance history", zap.Error(err))
		http.Error(w, "failed to insert balance history", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		h.logger.Error("failed to commit transaction", zap.Error(err))
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	h.broadcastBalanceUpdate(userID, info)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *PointsHandler) AddPointsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.logger.Error("user_id is required")
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	var req struct {
		Amount int `json:"amount"`
		Reason int `json:"reason_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request body", zap.Error(err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		h.logger.Error("balance adding amount must be positive")
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	tx, err := h.pg.Master.Begin(ctx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	var customerID int
	if err = tx.QueryRow(ctx, "SELECT id FROM customers WHERE id = $1", userID).Scan(&customerID); err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Error("user not found")
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to fetch points", zap.Error(err))
		http.Error(w, "failed to fetch points", http.StatusInternalServerError)
		return
	}

	info := models.PointsInfo{UserID: userID}
	if err = tx.QueryRow(ctx,
		`SELECT l.balance, ll.name 
		FROM loyalty l 
		JOIN loyalty_levels ll ON l.level_id = ll.id 
		WHERE customer_id = $1 FOR UPDATE`, userID,
	).Scan(&info.Balance, &info.Level); err != nil {
		if err == pgx.ErrNoRows {

			if _, err = tx.Exec(ctx, "INSERT INTO loyalty (customer_id, balance, level_id) VALUES ($1, $2, $3)", userID, 0, 1); err != nil {
				h.logger.Error("failed to create loyalty row", zap.Error(err))
				http.Error(w, "failed to create loyalty", http.StatusInternalServerError)
				return
			}
			info.Balance = 0
			info.Level = "Bronze" // default level
		} else {
			h.logger.Error("failed to fetch points", zap.Error(err))
			http.Error(w, "failed to fetch points", http.StatusInternalServerError)
			return
		}
	}

	if _, err = tx.Exec(ctx, "UPDATE loyalty SET balance = balance + $1 WHERE customer_id = $2", req.Amount, userID); err != nil {
		h.logger.Error("failed to update points", zap.Error(err))
		http.Error(w, "failed to update balance", http.StatusInternalServerError)
		return
	}

	if _, err = tx.Exec(ctx, "INSERT INTO balance_history (customer_id, points, created_at, reason_id) VALUES ($1, $2, $3, $4)", userID, req.Amount, time.Now(), req.Reason); err != nil {
		h.logger.Error("failed to insert balance history", zap.Error(err))
		http.Error(w, "failed to insert balance history", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		h.logger.Error("failed to commit transaction", zap.Error(err))
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	h.broadcastBalanceUpdate(userID, info)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *PointsHandler) broadcastBalanceUpdate(userID string, info models.PointsInfo) {
	messageBytes, err := json.Marshal(info)
	if err != nil {
		h.logger.Error("failed to marshal message", zap.Error(err))
		return
	}
	// sending message into websocket
	h.ws.Broadcast(messageBytes)
}
