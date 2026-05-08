# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependencies
COPY shared ./shared
COPY services/auth-service ./services/auth-service

WORKDIR /app/services/auth-service

# Setup module replacement
RUN go mod edit -replace shared=../../shared
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o main ./cmd/server

# Final stage
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/services/auth-service/main .
COPY --from=builder /app/services/auth-service/configs ./configs

# Copy environment file

EXPOSE 8081 9091

CMD ["./main"]