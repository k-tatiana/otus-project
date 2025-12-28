package external

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Intervals struct {
	MarketingCron   time.Duration `envconfig:"MARKETING_CRON" default:"1m"`
	OrdersCron      time.Duration `envconfig:"ORDERS_CRON" default:"1m"`
	UsingPointsCron time.Duration `envconfig:"USING_POINTS_CRON" default:"1m"`
}

type Config struct {
	BaseURL        string  `envconfig:"BASE_URL" required:"true"`
	SessionID      string  `envconfig:"SESSION_ID" required:"true"`
	DB             *string `envconfig:"DATABASE_URL" required:"true"`
	RMQ            string  `envconfig:"RABBITMQ_URL"`
	ServerAddr     string  `envconfig:"SERVER_ADDR" default:":8081"`
	QueueAvailable bool    `envconfig:"QUEUE_AVAILABLE" default:"true"`
	Intervals      Intervals
}

func NewConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
