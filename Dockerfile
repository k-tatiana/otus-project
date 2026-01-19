# Build stage
FROM golang:1.25-trixie AS builder

WORKDIR /app
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o otus-project ./cmd/otus-project

# Final stage
FROM alpine:3.19

RUN apk add --no-cache curl

# Copy binary from builder
COPY --from=builder /app/otus-project /app/otus-project

# Set environment variables for performance tuning
ENV GOMAXPROCS=2
ENV GOGC=100

WORKDIR /app
CMD ["./otus-project"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s \
  CMD curl -f http://localhost:8080/health || exit 1
