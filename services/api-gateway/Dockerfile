# =============================================================================
# Build Stage - Compile the Go application
# =============================================================================
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

WORKDIR /build

# Copy shared module first
COPY shared/ shared/

# Copy api-gateway module
COPY services/api-gateway/ api-gateway/

# Set working directory to api-gateway
WORKDIR /build/api-gateway

# Replace the shared module path to use local copy
RUN go mod edit -replace shared=../shared

# Download dependencies (cached if go.mod/go.sum unchanged)
RUN go mod download

# Build the binary with optimizations
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a \
    -installsuffix cgo \
    -o /bin/api-gateway \
    ./cmd/server


# =============================================================================
# Final Stage - Minimal runtime image
# =============================================================================
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    wget

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /bin/api-gateway .

# Copy configs directory
COPY --from=builder /build/api-gateway/configs ./configs

# Create empty .env file (environment variables will be provided by docker-compose/kubernetes)
RUN touch .env

# Set proper ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose application port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s \
    --timeout=3s \
    --start-period=5s \
    --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./api-gateway"]
