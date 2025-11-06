FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependencies
COPY shared ./shared
COPY services/location-service ./services/location-service

WORKDIR /app/services/location-service

# Setup module replacement
RUN go mod edit -replace shared=../../shared
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/services/location-service/main .
COPY --from=builder /app/services/location-service/data ./data

# Copy environment file
COPY services/location-service/.env .env

EXPOSE 8090 9092

CMD ["./main"]