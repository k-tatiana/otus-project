package middlewares

import (
	"net/http"

	"github.com/k-tatiana/otus-project/internal/services"
	"github.com/k-tatiana/otus-project/models"
)

// AuthMiddleware checks that the request has a valid session cookie.
// If not, it returns 401 Unauthorized.
func AuthMiddleware(store *services.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.Header.Get("X-Session-Id") == models.OrdersSessionID {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie("session_id")
			if err != nil || cookie.Value == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if _, ok := store.Get(r.Context(), cookie.Value); !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
