FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependencies
COPY shared ./shared
COPY services/message-service ./services/message-service

WORKDIR /app/services/message-service

# Setup module replacement
RUN go mod edit -replace echo-backend/shared=../../shared
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/services/message-service/main .

# Copy environment file
COPY services/message-service/.env .env

EXPOSE 8083 9093

CMD ["./main"]