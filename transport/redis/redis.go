package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Client struct {
	client *redis.Client
	logger *zap.Logger
}

const RedisTTL = 1 * time.Hour

func New(ctx context.Context, redisURL string, logger *zap.Logger) (*Client, error) {
	if redisURL == "" {
		return nil, errors.New("redis URL is required")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("failed to parse redis URL", zap.Error(err))
		return nil, err
	}

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", zap.Error(err))
		return nil, err
	}

	logger.Info("connected to redis", zap.String("url", redisURL))

	return &Client{
		client: rdb,
		logger: logger,
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return c.client.HGetAll(ctx, key)
}
