package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   prometheus.CounterVec
	HTTPRequestDuration prometheus.HistogramVec
	HTTPRequestSize     prometheus.HistogramVec
	HTTPResponseSize    prometheus.HistogramVec

	// Database metrics
	DBConnectionsActive prometheus.GaugeVec
	DBQueryDuration     prometheus.HistogramVec
	DBQueryErrors       prometheus.CounterVec

	// Redis metrics
	RedisCommandDuration prometheus.HistogramVec
	RedisCommandErrors   prometheus.CounterVec

	// Application metrics
	PointsUsed           prometheus.CounterVec
	PointsAdded          prometheus.CounterVec
	ActiveSessions       prometheus.GaugeVec
	WebSocketConnections prometheus.GaugeVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		HTTPRequestDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		HTTPRequestSize: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		// Database metrics
		DBConnectionsActive: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
			[]string{"database"},
		),
		DBQueryDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"query_type", "table"},
		),
		DBQueryErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"query_type", "table"},
		),

		// Redis metrics
		RedisCommandDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_command_duration_seconds",
				Help:    "Redis command latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"command"},
		),
		RedisCommandErrors: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_command_errors_total",
				Help: "Total number of Redis command errors",
			},
			[]string{"command"},
		),

		// Application metrics
		PointsUsed: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "points_used_total",
				Help: "Total number of points used",
			},
			[]string{"user_id"},
		),
		PointsAdded: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "points_added_total",
				Help: "Total number of points added",
			},
			[]string{"user_id"},
		),
		ActiveSessions: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "active_sessions",
				Help: "Number of active user sessions",
			},
			[]string{"status"},
		),
		WebSocketConnections: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "websocket_connections",
				Help: "Number of active WebSocket connections",
			},
			[]string{"status"},
		),
	}
}
