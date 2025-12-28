# Prometheus Metrics

This application exports Prometheus metrics for monitoring and observability.

## Available Metrics

### HTTP Metrics
- `http_requests_total` - Total number of HTTP requests (labels: method, endpoint, status)
- `http_request_duration_seconds` - HTTP request latency in seconds (labels: method, endpoint)
- `http_request_size_bytes` - HTTP request size in bytes (labels: method, endpoint)
- `http_response_size_bytes` - HTTP response size in bytes (labels: method, endpoint)

### Database Metrics
- `db_connections_active` - Number of active database connections (labels: database)
- `db_query_duration_seconds` - Database query latency in seconds (labels: query_type, table)
- `db_query_errors_total` - Total number of database query errors (labels: query_type, table)

### Redis Metrics
- `redis_command_duration_seconds` - Redis command latency in seconds (labels: command)
- `redis_command_errors_total` - Total number of Redis command errors (labels: command)

### Application Metrics
- `points_used_total` - Total number of points used (labels: user_id)
- `points_added_total` - Total number of points added (labels: user_id)
- `active_sessions` - Number of active user sessions (labels: status)
- `websocket_connections` - Number of active WebSocket connections (labels: status)

## Accessing Metrics

### Direct Access
Metrics are exposed at: `http://localhost:8100/metrics`

### Prometheus UI
When running with Docker Compose, Prometheus is available at: `http://localhost:9090`

## Docker Compose

The `prometheus` service is included in `docker-compose.yml` and will:
- Automatically scrape metrics from the server every 5 seconds
- Store time-series data in `./volumes/prometheus_data`
- Be accessible on port 9090

To start all services including Prometheus:
```bash
docker-compose up -d
```

## Configuration

Prometheus configuration is defined in `prometheus.yml`:
- Scrape interval: 5 seconds for the otus-server job
- Global evaluation interval: 15 seconds
- Metrics path: `/metrics`

## Querying Metrics

Example PromQL queries:
```
# Request rate (requests per second)
rate(http_requests_total[5m])

# Average request latency
avg(http_request_duration_seconds_bucket)

# Active sessions
active_sessions

# Points usage by user
rate(points_used_total[5m])
```
