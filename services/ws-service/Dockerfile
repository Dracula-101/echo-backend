# Multi-stage build for production

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go workspace files
COPY go.work go.work.sum ./

# Copy service files
COPY services/ws-service /app/services/ws-service
COPY shared /app/shared

# Build the application
WORKDIR /app/services/ws-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/ws-service cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/services/ws-service/bin/ws-service .
COPY --from=builder /app/services/ws-service/configs ./configs

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8087

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8087/health || exit 1

# Run the application
CMD ["./ws-service"]
