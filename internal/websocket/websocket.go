package websocket

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/k-tatiana/otus-project/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Hub maintains active client connections and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
}

// Client represents a WebSocket connection.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	sessionID string
	userID    string
}

// WebSocketHandler handles WebSocket connections.
type WebSocketHandler struct {
	hub          *Hub
	sessionStore *services.SessionStore
	logger       *zap.Logger
}

// NewWebSocketHandler creates a new WebSocket handler.
func NewWebSocketHandler(sessionStore *services.SessionStore, logger *zap.Logger) *WebSocketHandler {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}

	go hub.run()

	return &WebSocketHandler{
		hub:          hub,
		sessionStore: sessionStore,
		logger:       logger,
	}
}

// ServeWS handles WebSocket upgrade and connection.
func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("session_id")
	if sessionID == "" {
		h.logger.Error("missing session_id header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, ok := h.sessionStore.Get(r.Context(), sessionID)
	if !ok {
		h.logger.Error("invalid session", zap.String("session_id", sessionID))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		hub:       h.hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		sessionID: sessionID,
		userID:    strconv.Itoa(user.ID),
	}

	h.hub.register <- client

	go client.readPump()
	go client.writePump()

	h.logger.Info("WebSocket client connected", zap.String("user_id", client.userID))
}

// Broadcast sends a message to all connected clients.
func (h *WebSocketHandler) Broadcast(message []byte) {
	h.hub.broadcast <- message
}

// run manages client registration, unregistration, and message broadcasting.
func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client] = true
			hub.mu.Unlock()
			hub.logger.Debug("client registered", zap.String("user_id", client.userID))

		case client := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
				hub.mu.Unlock()
				hub.logger.Debug("client unregistered", zap.String("user_id", client.userID))
			} else {
				hub.mu.Unlock()
			}

		case message := <-hub.broadcast:
			hub.mu.RLock()
			for client := range hub.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(hub.clients, client)
				}
			}
			hub.mu.RUnlock()
		}
	}
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("websocket error", zap.Error(err))
			}
			break
		}

		c.hub.broadcast <- message
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
