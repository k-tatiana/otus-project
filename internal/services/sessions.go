package services

import (
	"context"
	"sync"
	"time"

	"github.com/k-tatiana/otus-project/models"
	"github.com/k-tatiana/otus-project/transport/redis"
)

const sessionStoreTTL = 24 * time.Hour

// SessionStore is an in-memory mock session storage.
type SessionStore struct {
	redis *redis.Client
	mu    sync.Mutex
}

func NewSessionStore(redis *redis.Client) *SessionStore {
	return &SessionStore{
		redis: redis,
	}
}

func (s *SessionStore) Set(ctx context.Context, sessionID string, user models.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.redis.Set(ctx, sessionID, user, sessionStoreTTL)
	if err != nil {
		return
	}
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (models.User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var user models.User
	getResult := s.redis.HGetAll(ctx, sessionID)
	if getResult.Err() != nil {
		return models.User{}, false
	}
	err := getResult.Scan(&user)
	if err != nil {
		return models.User{}, false
	}
	return user, true
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redis.Del(ctx, sessionID)
}
