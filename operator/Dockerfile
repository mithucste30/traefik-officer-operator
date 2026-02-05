# Dockerfile for traefik-officer-operator
# Builds the operator binary with embedded log processing

FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Copy root go.mod for shared code
COPY go.mod go.sum ./
RUN go mod download

# Copy shared code and pkg (needed for log processing)
COPY shared/ ./shared/
COPY pkg/ ./pkg/

# Copy operator go.mod
COPY operator/go.mod operator/go.sum ./
RUN go mod download

# Copy operator source
COPY operator/ ./

# Build operator binary
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
