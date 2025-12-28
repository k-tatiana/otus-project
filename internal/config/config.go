package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port              string `envconfig:"PORT" default:"8080"`
	MasterDatabaseURL string `envconfig:"MASTER_DATABASE_URL" required:"true"`
	EnableSlave       bool   `envconfig:"ENABLE_SLAVE" default:"false"`
	SlaveDatabaseURL  string `envconfig:"SLAVE_DATABASE_URL"`
	RedisURL          string `envconfig:"REDIS_URL" default:"redis://localhost:6379"`
	RabbitMQURL       string `envconfig:"RABBITMQ_URL" default:"amqp://guest:guest@localhost:5672/"`
	SessionKey        string `envconfig:"SESSION_KEY" default:"secret-key"`
	OAuthClientID     string `envconfig:"OAUTH_CLIENT_ID"`
	OAuthClientSecret string `envconfig:"OAUTH_CLIENT_SECRET"`
}

// NewConfig loads configuration from environment variables.
func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// MustLoadConfig loads configuration and panics on error.
func MustLoadConfig() *Config {
	cfg, err := NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return cfg
}
