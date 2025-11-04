FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependencies
COPY shared ./shared
COPY services/api-gateway ./services/api-gateway

WORKDIR /app/services/api-gateway

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
COPY --from=builder /app/services/api-gateway/main .
COPY --from=builder /app/services/api-gateway/configs ./configs
COPY --from=builder /app/services/api-gateway/tls ./tls

# Copy environment file
COPY services/api-gateway/.env .env

EXPOSE 8080 9090

CMD ["./main"]