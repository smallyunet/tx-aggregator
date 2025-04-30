# Build stage
FROM golang:1.23.2 AS builder

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary from cmd/tx-aggregator
RUN CGO_ENABLED=0 GOOS=linux go build -o tx-aggregator ./cmd/tx-aggregator

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary only
COPY --from=builder /app/tx-aggregator .

# Create logs directory (for file logging)
RUN mkdir -p /app/logs

# Default user is root (no USER directive)

# Expose application port
EXPOSE 8080

# Declare config volume (optional but declarative)
VOLUME ["/app/config"]

# Entry point
ENTRYPOINT ["./tx-aggregator"]
