FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependencies
COPY shared ./shared
COPY services/auth-service ./services/auth-service

WORKDIR /app/services/auth-service

# Setup module replacement
RUN go mod edit -replace shared=../../shared
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/services/auth-service/main .
COPY --from=builder /app/services/auth-service/configs ./configs

# Copy environment file
COPY services/auth-service/.env .env

EXPOSE 8081 9091

CMD ["./main"]