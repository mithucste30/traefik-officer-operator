# Multi-stage Dockerfile for Traefik Officer Operator
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git make gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
COPY operator/go.mod operator/go.sum ./operator/ 2>/dev/null || true

# Download dependencies
RUN go mod download

# Copy source code
COPY pkg/ ./pkg/
COPY operator/ ./operator/

# Build operator binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/bin/traefik-officer-operator ./operator/main.go

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /usr/bin/traefik-officer-operator /usr/local/bin/traefik-officer-operator

# Create non-root user
RUN adduser -D appuser && \
    chown -R appuser:appuser /home/appuser

USER appuser

WORKDIR /home/appuser

# Expose ports
EXPOSE 8080 8081 9443

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/traefik-officer-operator"]
