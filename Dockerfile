# Multi-stage Dockerfile for traefik-officer
# Builds both standalone and operator binaries

FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared code and pkg
COPY shared/ ./shared/
COPY pkg/ ./pkg/

# Copy cmd directory
COPY cmd/ ./cmd/

# Build standalone binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /traefik-officer ./cmd/traefik-officer

# Final stage for standalone binary
FROM alpine:3.19 AS standalone

RUN apk add --no-cache ca-certificates

COPY --from=builder /traefik-officer /app/traefik-officer

WORKDIR /app

RUN adduser -D appuser && chown -R appuser /app
USER appuser

ENTRYPOINT ["/app/traefik-officer"]
