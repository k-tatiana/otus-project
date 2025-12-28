FROM golang:1.25-trixie

WORKDIR /app
COPY . .
RUN go build -o otus-project ./cmd/otus-project
CMD ["./otus-project"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s CMD curl -f http://localhost:8080/health || exit 1
