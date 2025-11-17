# Echo Backend Usage Guide

Complete developer guide for working with the Echo Backend microservices platform.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Make Commands Reference](#make-commands-reference)
- [Service Management](#service-management)
- [Database Operations](#database-operations)
- [API Testing](#api-testing)
- [WebSocket Testing](#websocket-testing)
- [Debugging](#debugging)
- [Common Workflows](#common-workflows)
- [Troubleshooting](#troubleshooting)

## Getting Started

### Prerequisites Installation

**macOS**:
```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install go@1.21
brew install --cask docker
brew install make
```

**Linux (Ubuntu/Debian)**:
```bash
# Install Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Make
sudo apt-get update
sudo apt-get install build-essential
```

### Project Setup

```bash
# Clone repository
git clone https://github.com/yourusername/echo-backend.git
cd echo-backend

# Verify Go workspace
cat go.work  # Should list all services and shared modules

# Start all services
make up

# Verify services are running
make health
```

### First Time Setup

```bash
# Initialize database schemas
make db-init

# Run database migrations
make db-migrate

# Seed test data (optional)
make db-seed

# Check all services are healthy
make status
```

## Development Environment

### Directory Structure

```
echo-backend/
├── services/              # All microservices
│   ├── api-gateway/
│   │   ├── cmd/server/main.go       # Entry point
│   │   ├── internal/                # Service code
│   │   │   ├── config/             # Configuration
│   │   │   ├── handler/            # HTTP handlers
│   │   │   ├── service/            # Business logic
│   │   │   └── health/             # Health checks
│   │   ├── configs/                # YAML configs
│   │   │   ├── config.yaml         # Base config
│   │   │   ├── config.dev.yaml     # Dev overrides
│   │   │   └── config.prod.yaml    # Prod overrides
│   │   ├── Dockerfile              # Production build
│   │   ├── Dockerfile.dev          # Development build
│   │   └── go.mod                  # Service module
│   └── ... (other services)
├── shared/                # Shared libraries
│   ├── pkg/              # Infrastructure packages
│   └── server/           # HTTP utilities
├── database/             # Database files
│   └── schemas/          # SQL schemas
├── infra/                # Infrastructure
│   ├── docker/           # Docker Compose files
│   └── scripts/          # Utility scripts
├── Makefile              # Development commands
├── go.work               # Go workspace
└── .env                  # Environment variables
```

### Environment Variables

Create a `.env` file in the root directory:

```bash
# Application
APP_ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=echo
DB_USER=echo
DB_PASSWORD=echo
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=echo-backend

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h  # 7 days

# Services
API_GATEWAY_PORT=8080
AUTH_SERVICE_PORT=8081
USER_SERVICE_PORT=8082
MESSAGE_SERVICE_PORT=8083
PRESENCE_SERVICE_PORT=8084

# Logging
LOG_LEVEL=debug
LOG_FORMAT=console  # or json
```

### Hot Reload

Services automatically reload when code changes (in dev mode):

```bash
# Start services with hot reload enabled
ENV=dev make up

# Watch logs to see reload events
make auth-logs
```

## Make Commands Reference

### Service Lifecycle

```bash
# Start all services
make up

# Start in production mode
ENV=prod make up

# Stop all services
make down

# Rebuild and restart all services
make rerun

# View logs from all services
make logs

# Follow logs (tail -f)
make logs-follow

# Check service status
make status

# Run health checks
make health
```

### Individual Service Commands

Pattern: `make <service>-<action>`

```bash
# Start specific service
make auth-up
make message-up
make gateway-up

# Stop specific service
make auth-down
make message-down

# Restart specific service
make auth-rerun
make message-rerun

# View service logs
make auth-logs
make message-logs

# Follow service logs
make auth-logs-follow

# Rebuild service (no cache)
make auth-rebuild
make message-rebuild

# Shell into service container
make auth-shell
make message-shell
```

### Database Commands

```bash
# Initialize database schemas
make db-init

# Run all migrations
make db-migrate

# Rollback last migration
make db-migrate-down

# Show migration status
make db-migrate-status

# Connect to PostgreSQL CLI
make db-connect

# Seed test data
make db-seed

# Reset database (DANGER: drops all data)
make db-reset

# Backup database
make db-backup

# Restore database from backup
make db-restore
```

### Infrastructure Commands

```bash
# List Kafka topics
make kafka-topics

# Create Kafka topics
make kafka-create-topics

# Connect to Kafka console consumer
make kafka-console-consumer TOPIC=user.registered

# Connect to Redis CLI
make redis-connect

# View Redis keys
make redis-keys

# Flush Redis (DANGER: deletes all cache)
make redis-flush
```

### Testing Commands

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Test specific service
cd services/auth-service && go test -v ./...

# Integration tests
make test-integration

# Test auth endpoints
make test-auth

# Test message endpoints
make test-message

# Verify security
make verify-security
```

### Development Tools

```bash
# Format all code
make fmt

# Run linters
make lint

# Run security checks
make security-check

# Generate mocks
make generate-mocks

# Update dependencies
make deps-update

# Tidy dependencies
make deps-tidy

# View dependency tree
make deps-tree
```

## Service Management

### Starting Individual Services

**Option 1: Via Make**
```bash
make auth-up
```

**Option 2: Via Docker Compose**
```bash
docker compose -f infra/docker/docker-compose.dev.yml up auth-service
```

**Option 3: Via Go (without Docker)**
```bash
cd services/auth-service
go run cmd/server/main.go
```

### Viewing Service Logs

**All Services**:
```bash
make logs
```

**Specific Service**:
```bash
make auth-logs
make message-logs
```

**Follow Logs** (like tail -f):
```bash
make auth-logs-follow
```

**Filter Logs**:
```bash
# Show only errors
docker logs echo-auth-service 2>&1 | grep "ERROR"

# Show requests with slow response times
docker logs echo-auth-service 2>&1 | grep "duration_ms" | awk '$NF > 100'
```

### Service Health Checks

**Check All Services**:
```bash
make health
```

**Check Specific Service**:
```bash
# Liveness check
curl http://localhost:8081/health

# Readiness check
curl http://localhost:8081/ready
```

**Expected Response**:
```json
{
  "status": "up",
  "timestamp": "2024-01-15T10:30:45Z",
  "service": "auth-service",
  "version": "1.0.0",
  "checks": {
    "database": {
      "status": "up",
      "duration_ms": 5,
      "message": "Connected"
    },
    "cache": {
      "status": "up",
      "duration_ms": 2,
      "message": "Connected"
    }
  }
}
```

## Database Operations

### Connecting to Database

```bash
# Via Make
make db-connect

# Via psql directly
psql -h localhost -p 5432 -U echo -d echo

# Via Docker
docker exec -it echo-postgres psql -U echo -d echo
```

### Running Queries

```sql
-- View all schemas
\dn

-- Switch to auth schema
SET search_path TO auth;

-- List tables
\dt

-- View users
SELECT id, phone, verified, created_at FROM auth.users;

-- View sessions
SELECT user_id, device_id, ip_address, created_at
FROM auth.sessions
WHERE expires_at > NOW();

-- View recent messages
SELECT id, from_user_id, to_user_id, content, created_at
FROM messages.messages
ORDER BY created_at DESC
LIMIT 10;
```

### Database Migrations

**Create New Migration**:
```bash
# Create migration files
cd database/schemas/auth
touch $(date +%Y%m%d)_add_user_roles.up.sql
touch $(date +%Y%m%d)_add_user_roles.down.sql
```

**Up Migration** (`*_up.sql`):
```sql
-- Add new column
ALTER TABLE auth.users ADD COLUMN role VARCHAR(50) DEFAULT 'user';

-- Create index
CREATE INDEX idx_users_role ON auth.users(role) WHERE deleted_at IS NULL;
```

**Down Migration** (`*_down.sql`):
```sql
-- Remove index
DROP INDEX IF EXISTS auth.idx_users_role;

-- Remove column
ALTER TABLE auth.users DROP COLUMN IF EXISTS role;
```

**Run Migration**:
```bash
make db-migrate
```

**Rollback Migration**:
```bash
make db-migrate-down
```

### Database Seeding

**Edit Seed Script**:
```bash
vim infra/scripts/seed-db.sh
```

**Add Seed Data**:
```sql
-- Insert test users
INSERT INTO auth.users (id, phone, password_hash, verified)
VALUES
  (gen_random_uuid(), '+1234567890', '$argon2id$...', true),
  (gen_random_uuid(), '+9876543210', '$argon2id$...', true);

-- Insert test profiles
INSERT INTO users.profiles (user_id, display_name, bio)
VALUES
  ((SELECT id FROM auth.users WHERE phone = '+1234567890'), 'John Doe', 'Test user 1'),
  ((SELECT id FROM auth.users WHERE phone = '+9876543210'), 'Jane Smith', 'Test user 2');
```

**Run Seed**:
```bash
make db-seed
```

## API Testing

### Using cURL

**Register User**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+1234567890",
    "password": "SecurePass123!",
    "name": "John Doe"
  }'
```

**Verify OTP**:
```bash
# Get OTP from logs or database
docker logs echo-auth-service 2>&1 | grep "OTP:"

curl -X POST http://localhost:8080/api/v1/auth/verify-otp \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+1234567890",
    "otp": "123456"
  }'
```

**Login**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+1234567890",
    "password": "SecurePass123!"
  }' | jq '.'
```

**Store Token**:
```bash
# Extract token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"phone": "+1234567890", "password": "SecurePass123!"}' \
  | jq -r '.data.access_token')

echo $TOKEN
```

**Authenticated Request**:
```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.'
```

**Send Message**:
```bash
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to_user_id": "recipient-uuid",
    "content": "Hello!",
    "type": "text"
  }' | jq '.'
```

### Using HTTPie

```bash
# Install HTTPie
brew install httpie  # macOS
apt install httpie   # Linux

# Register
http POST localhost:8080/api/v1/auth/register \
  phone="+1234567890" \
  password="SecurePass123!" \
  name="John Doe"

# Login
http POST localhost:8080/api/v1/auth/login \
  phone="+1234567890" \
  password="SecurePass123!"

# Authenticated request
http GET localhost:8080/api/v1/users/profile \
  Authorization:"Bearer $TOKEN"
```

### Using Postman

1. Import collection: `docs/postman/echo-backend.json`
2. Set environment variables:
   - `base_url`: `http://localhost:8080`
   - `token`: (will be set automatically after login)
3. Run authentication flow
4. Test endpoints

## WebSocket Testing

### Using wscat

```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c "ws://localhost:8083/ws" \
  -H "X-User-ID: your-user-uuid" \
  -H "X-Device-ID: device-1" \
  -H "X-Platform: web"

# Send message
> {"type": "message", "content": "Hello!"}

# Receive messages
< {"type": "message", "from": "uuid", "content": "Hi there!"}
```

### Using Browser Console

```javascript
// Connect
const ws = new WebSocket('ws://localhost:8083/ws');

// Set headers (must be done before connection)
// Note: WebSocket in browser doesn't support custom headers
// Use query parameters instead:
const userId = 'your-user-uuid';
const deviceId = 'device-1';
const platform = 'web';
const ws = new WebSocket(`ws://localhost:8083/ws?user_id=${userId}&device_id=${deviceId}&platform=${platform}`);

// Handle connection open
ws.onopen = () => {
  console.log('Connected');

  // Send message
  ws.send(JSON.stringify({
    type: 'message',
    content: 'Hello from browser!'
  }));
};

// Handle incoming messages
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};

// Handle errors
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

// Handle close
ws.onclose = () => {
  console.log('Disconnected');
};
```

### Using Python

```python
#!/usr/bin/env python3
import asyncio
import websockets
import json

async def test_websocket():
    uri = "ws://localhost:8083/ws"
    headers = {
        "X-User-ID": "your-user-uuid",
        "X-Device-ID": "device-1",
        "X-Platform": "web"
    }

    async with websockets.connect(uri, extra_headers=headers) as websocket:
        # Send message
        await websocket.send(json.dumps({
            "type": "message",
            "content": "Hello from Python!"
        }))

        # Receive messages
        while True:
            message = await websocket.recv()
            data = json.loads(message)
            print(f"Received: {data}")

asyncio.run(test_websocket())
```

## Debugging

### Viewing Logs

**Structured Logs** (JSON format in production):
```bash
docker logs echo-auth-service --tail 100 -f | jq '.'
```

**Filter by Log Level**:
```bash
# Errors only
docker logs echo-auth-service 2>&1 | jq 'select(.level == "error")'

# Warnings and errors
docker logs echo-auth-service 2>&1 | jq 'select(.level == "warn" or .level == "error")'
```

**Filter by Request ID**:
```bash
docker logs echo-auth-service 2>&1 | jq 'select(.request_id == "req_abc123")'
```

**Filter by Duration**:
```bash
# Requests taking > 100ms
docker logs echo-auth-service 2>&1 | jq 'select(.duration_ms > 100)'
```

### Debugging with Delve

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug specific service
cd services/auth-service
dlv debug cmd/server/main.go

# Set breakpoint
(dlv) break main.main
(dlv) break internal/handler/auth.go:45

# Continue execution
(dlv) continue

# Inspect variables
(dlv) print user
(dlv) print request

# Step through code
(dlv) next
(dlv) step
(dlv) stepout
```

### Database Debugging

**Enable Query Logging**:
```yaml
# config.dev.yaml
database:
  log_queries: true
  log_level: debug
```

**View Slow Queries**:
```sql
-- Enable slow query logging
ALTER DATABASE echo SET log_min_duration_statement = 100; -- 100ms

-- View slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;
```

**View Active Connections**:
```sql
SELECT pid, usename, application_name, client_addr, state, query
FROM pg_stat_activity
WHERE datname = 'echo';
```

### Redis Debugging

```bash
# Connect to Redis
make redis-connect

# Monitor all commands
redis-cli MONITOR

# View all keys
redis-cli KEYS "*"

# View session keys
redis-cli KEYS "session:*"

# Get key value
redis-cli GET "session:abc123"

# View key TTL
redis-cli TTL "session:abc123"
```

### Kafka Debugging

```bash
# List topics
make kafka-topics

# Consume messages from topic
docker exec -it echo-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic user.registered \
  --from-beginning

# Produce test message
docker exec -it echo-kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic user.registered
> {"user_id": "uuid", "phone": "+1234567890"}

# View consumer groups
docker exec -it echo-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --list

# View consumer group lag
docker exec -it echo-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --group echo-backend \
  --describe
```

## Common Workflows

### Adding a New Endpoint

1. **Define Route** in handler:
```go
// internal/handler/user.go
func (h *UserHandler) SetupRoutes(r *router.Router) {
    r.GET("/api/v1/users/:id", h.GetUser)
    r.PUT("/api/v1/users/:id", h.UpdateUser)
    r.DELETE("/api/v1/users/:id", h.DeleteUser)
}
```

2. **Implement Handler**:
```go
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := router.Param(r, "id")

    user, err := h.service.GetUser(r.Context(), userID)
    if err != nil {
        response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error())
        return
    }

    response.OK(w, user)
}
```

3. **Implement Service Logic**:
```go
// internal/service/user.go
func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    user, err := s.repo.FindByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}
```

4. **Register Route** in main.go:
```go
userHandler := handler.NewUserHandler(userService, log)

router := router.NewBuilder().
    WithRoutes(func(r *router.Router) {
        userHandler.SetupRoutes(r)
    }).
    Build()
```

5. **Test Endpoint**:
```bash
curl http://localhost:8080/api/v1/users/uuid | jq '.'
```

### Adding a New Service

See [GUIDELINES.md](./GUIDELINES.md#adding-a-new-service) for detailed steps.

### Updating Configuration

1. **Update Config Struct**:
```go
// internal/config/config.go
type Config struct {
    // ... existing fields
    NewFeature NewFeatureConfig `mapstructure:"new_feature"`
}

type NewFeatureConfig struct {
    Enabled bool   `mapstructure:"enabled"`
    Timeout int    `mapstructure:"timeout"`
}
```

2. **Update YAML Config**:
```yaml
# configs/config.yaml
new_feature:
  enabled: ${NEW_FEATURE_ENABLED:false}
  timeout: ${NEW_FEATURE_TIMEOUT:30}
```

3. **Add Validation**:
```go
// internal/config/validator.go
func (c *Config) ValidateAndSetDefaults() error {
    // ... existing validations
    if c.NewFeature.Timeout <= 0 {
        c.NewFeature.Timeout = 30
    }
    return nil
}
```

4. **Update Environment**:
```bash
# .env
NEW_FEATURE_ENABLED=true
NEW_FEATURE_TIMEOUT=60
```

5. **Restart Service**:
```bash
make auth-rerun
```

## Troubleshooting

### Services Not Starting

**Check Docker Status**:
```bash
docker ps
docker compose -f infra/docker/docker-compose.dev.yml ps
```

**Check Logs**:
```bash
make logs
```

**Common Issues**:
- Port already in use: `lsof -i :8080` and kill process
- Docker daemon not running: `docker info`
- Missing environment variables: check `.env` file

### Database Connection Errors

**Test Connection**:
```bash
psql -h localhost -p 5432 -U echo -d echo -c "SELECT 1"
```

**Check PostgreSQL Status**:
```bash
docker ps | grep postgres
docker logs echo-postgres
```

**Reset Database**:
```bash
make db-reset
```

### Redis Connection Errors

**Test Connection**:
```bash
redis-cli -h localhost -p 6379 ping
```

**Check Redis Status**:
```bash
docker ps | grep redis
docker logs echo-redis
```

### Kafka Connection Errors

**Check Kafka Status**:
```bash
docker ps | grep kafka
docker logs echo-kafka
```

**Recreate Topics**:
```bash
make kafka-create-topics
```

### WebSocket Connection Refused

**Check Message Service**:
```bash
docker ps | grep message-service
docker logs echo-message-service
```

**Test WebSocket Endpoint**:
```bash
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
  http://localhost:8083/ws
```

### High Memory Usage

**Check Container Stats**:
```bash
docker stats
```

**Optimize Connection Pool**:
```yaml
# config.yaml
database:
  max_open_connections: 10  # Reduce from 25
  max_idle_connections: 5   # Reduce from 10
```

**Clear Cache**:
```bash
make redis-flush
```

### Slow Response Times

**Enable Query Logging**:
```yaml
database:
  log_queries: true
  log_level: debug
```

**Check Slow Queries**:
```bash
docker logs echo-auth-service 2>&1 | grep "duration_ms" | awk '$NF > 100'
```

**Add Database Indexes**:
```sql
CREATE INDEX idx_users_phone ON auth.users(phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_conversation ON messages.messages(conversation_id, created_at DESC);
```

---

For more information:
- [Architecture Documentation](./ARCHITECTURE.md)
- [Development Guidelines](./GUIDELINES.md)
- [Contributing Guide](./CONTRIBUTING.md)

**Last Updated**: January 2025
