# Dockerfile for traefik-officer-operator
# Builds the operator binary with embedded log processing
# This Dockerfile should be built from the repository root

FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Copy root module files first
COPY go.mod go.sum ./

# Copy shared code and pkg (needed for log processing and for replace directive to work)
COPY shared/ ./shared/
COPY pkg/ ./pkg/

# Copy operator module
COPY operator/ ./operator/

# Build operator binary from the operator directory
WORKDIR /app/operator
RUN CGO_ENABLED=0 GOOS=linux go build -o /traefik-officer-operator .

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /traefik-officer-operator /usr/local/bin/traefik-officer-operator

RUN adduser -D appuser
USER appuser

WORKDIR /home/appuser

EXPOSE 8080 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/traefik-officer-operator"]
