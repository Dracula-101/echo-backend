FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependencies
COPY shared/go.mod shared/go.sum ./shared/
COPY services/user-service/go.mod services/user-service/go.sum ./services/user-service/

WORKDIR /app/services/user-service

# Setup module replacement
RUN go mod edit -replace shared=../../shared
RUN go mod download

# Copy source code
COPY shared/ ../../shared/
COPY services/user-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/user-service cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/services/user-service/bin/user-service .
COPY --from=builder /app/services/user-service/configs ./configs

EXPOSE 8084

CMD ["./user-service"]
