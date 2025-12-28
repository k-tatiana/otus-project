package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	svc "github.com/k-tatiana/otus-project/internal/services"
	"github.com/k-tatiana/otus-project/models"
	"github.com/k-tatiana/otus-project/transport/postgres"
)

type AuthHandler struct {
	sessionStore *svc.SessionStore
	logger       *zap.Logger
	db           *postgres.DB
}

type AuthResponse struct {
	Message   string      `json:"message"`
	User      models.User `json:"user"`
	SessionID string      `json:"session_id"`
}

func NewAuthHandler(sessionStore *svc.SessionStore, logger *zap.Logger, db *postgres.DB) *AuthHandler {
	return &AuthHandler{
		sessionStore: sessionStore,
		logger:       logger,
		db:           db,
	}
}

// loginHandler simulates redirect to an OAuth provider authorization page.
// For the mock provider we immediately redirect back to our callback
// with a fixed "code" value as if the user successfully authorized.
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// In a real OAuth flow, here we would redirect to the provider's
	// authorization URL. For the mock, we immediately continue the flow.
	callbackURL := "/auth/callback?code=mock-code"
	http.Redirect(w, r, callbackURL, http.StatusFound)
}

// callbackHandler finishes the mock OAuth flow, creates a session and
// sets a session cookie for the user.
func (h *AuthHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.URL.Query().Get("code")
	if code != "mock-code" {
		http.Error(w, "invalid authorization code", http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("user_id")
	if userID == "" {
		h.logger.Error("user_id header is missing")
		http.Error(w, "user_id header is required", http.StatusBadRequest)
		return
	}
	h.logger.Info("User ID from header", zap.String("user_id", userID))

	// Convert userID to integer (assuming it's a string representation of an ID)
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		h.logger.Error("invalid user_id format", zap.Error(err))
		http.Error(w, "invalid user_id format", http.StatusBadRequest)
		return
	}

	var user models.User
	row := h.db.Slave.QueryRow(r.Context(), "SELECT first_name, last_name FROM customers WHERE id = $1", userIDInt)
	if row == nil {
		h.logger.Error("failed to query user: row is nil")
		http.Error(w, "failed to query user", http.StatusInternalServerError)
		return
	}
	if err := row.Scan(&user.FirstName, &user.LastName); err != nil {
		h.logger.Error("failed to query user", zap.Error(err))
		http.Error(w, "failed to query user", http.StatusInternalServerError)
		return
	}

	sessionID, err := newSessionID()
	if err != nil {
		h.logger.Error("failed to generate session ID", zap.Error(err))
		http.Error(w, "failed to generate session ID", http.StatusInternalServerError)
		return
	}

	h.sessionStore.Set(ctx, sessionID, user)

	http.SetCookie(w, &http.Cookie{
		Name:        "session_id",
		Value:       sessionID,
		Path:        "/",
		HttpOnly:    false,
		Secure:      false, // for local development only
		SameSite:    http.SameSiteLaxMode,
		Expires:     time.Now().Add(24 * time.Hour),
		Quoted:      false,
		Domain:      "",
		RawExpires:  "",
		MaxAge:      0,
		Partitioned: false,
		Raw:         "",
		Unparsed:    []string{},
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(AuthResponse{
		Message:   "mock login successful",
		User:      user,
		SessionID: sessionID,
	}); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// logoutHandler removes the user's session and clears the cookie.
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("session_id")
	if err == nil {
		h.sessionStore.Delete(ctx, cookie.Value)
	}

	expiredCookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, expiredCookie)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{
		Message: "logged out",
	}); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// newSessionID generates a random session identifier.
func newSessionID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
