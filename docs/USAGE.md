# Echo Backend Usage Guide

Complete developer guide for working with the Echo Backend microservices platform - based on **ACTUAL** implementation.

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
git clone https://github.com/Dracula-101/echo-backend.git
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
# Database is automatically initialized on first run via db-init container

# Check all services are healthy
make status

# View service logs
make logs
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
│   │   │   ├── proxy/              # Proxy manager
│   │   │   └── health/             # Health checks
│   │   ├── configs/                # YAML configs
│   │   │   ├── config.yaml         # Base config
│   │   │   ├── config.dev.yaml     # Dev overrides
│   │   │   ├── config.prod.yaml    # Prod overrides
│   │   │   └── routes.yaml         # Route configuration
│   │   ├── Dockerfile              # Production build
│   │   ├── Dockerfile.dev          # Development build (hot reload)
│   │   └── go.mod                  # Service module
│   ├── auth-service/       # Authentication (2 endpoints)
│   ├── message-service/    # Messaging + WebSocket (9 endpoints)
│   ├── user-service/       # User profiles (2 endpoints)
│   ├── location-service/   # IP geolocation (2 endpoints)
│   ├── presence-service/   # Presence tracking (stubbed)
│   ├── media-service/      # Media handling (placeholder)
│   ├── notification-service/  # Push notifications (placeholder)
│   └── analytics-service/  # Analytics (placeholder)
├── shared/                # Shared libraries
│   ├── pkg/              # Infrastructure packages
│   │   ├── database/    # PostgreSQL interface
│   │   ├── cache/       # Redis interface
│   │   ├── messaging/   # Kafka interface
│   │   └── logger/      # Structured logging
│   └── server/           # HTTP utilities
│       ├── router/      # Router builder
│       ├── middleware/  # 15+ middleware
│       ├── response/    # Standard responses
│       └── shutdown/    # Graceful shutdown
├── database/             # Database files
│   └── schemas/          # SQL schemas (auth, users, messages, etc.)
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

# PostgreSQL
POSTGRES_USER=echo
POSTGRES_PASSWORD=echo_password
POSTGRES_DB=echo_db

# Redis
REDIS_PASSWORD=redis_password

# JWT (configured in service config files)
# See services/auth-service/configs/config.yaml

# Services auto-configure ports via docker-compose
# API Gateway: 8080 (external)
# Auth Service: 8081 (internal)
# User Service: 8082 (internal)
# Message Service: 8083 (internal)
# Media Service: 8084 (internal)
# Presence Service: 8085 (internal)
# Location Service: 8090 (external)

# Logging
LOG_LEVEL=debug
LOG_FORMAT=console  # or json
```

### Hot Reload

Services automatically reload when code changes (in dev mode using Air):

```bash
# Services are started in dev mode by default
make up

# Watch logs to see reload events
make auth-logs
# You'll see: "Running..." when file changes are detected
```

## Make Commands Reference

### Service Lifecycle

```bash
# Start all services
make up

# Stop all services
make down

# Rebuild and restart all services
make rerun

# View logs from all services
make logs

# Check service status
make status

# Run health checks
make health
```

### Individual Service Commands

Pattern: `make <service>-<action>`

```bash
# Available services:
# - gateway (api-gateway)
# - auth (auth-service)
# - message (message-service)
# - user (user-service)
# - location (location-service)
# - presence (presence-service)
# - media (media-service)

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

# Rebuild service (no cache)
make auth-rebuild
make message-rebuild
```

### Database Commands

```bash
# Database is auto-initialized via db-init container

# Connect to PostgreSQL CLI
make db-connect

# Manual operations (if needed)
docker exec -it echo-postgres psql -U echo -d echo_db
```

### Infrastructure Commands

```bash
# Kafka operations
make kafka-topics

# Connect to Redis CLI
make redis-connect

# View Redis keys
docker exec -it echo-redis redis-cli KEYS "*"
```

### Testing Commands

```bash
# Run tests (when available)
make test

# Test specific service
cd services/auth-service && go test -v ./...

# Test auth endpoints
make test-auth
```

### Development Tools

```bash
# Format all code
make fmt

# Run linters
make lint

# Tidy dependencies
make deps-tidy
```

## Service Management

### Starting Individual Services

**Option 1: Via Make**
```bash
make auth-up
```

**Option 2: Via Docker Compose**
```bash
docker compose -f infra/docker/docker-compose.dev.yml up -d auth-service
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

**Follow Logs** (real-time):
```bash
docker logs -f echo-auth-service
docker logs -f echo-message-service
```

**Filter Logs**:
```bash
# Show only errors
docker logs echo-auth-service 2>&1 | grep "error"

# Show requests with slow response times (development console format)
docker logs echo-auth-service 2>&1 | grep "duration"
```

### Service Health Checks

**Check All Services**:
```bash
make health
```

**Check Specific Service**:
```bash
# API Gateway
curl http://localhost:8080/health

# Auth Service (via gateway or direct if exposed)
curl http://localhost:8081/health

# Message Service
curl http://localhost:8083/health

# Location Service
curl http://localhost:8090/health
```

**Expected Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:45Z",
  "service": "auth-service",
  "version": "1.0.0",
  "checks": {
    "database": {
      "status": "up",
      "latency_ms": 5
    },
    "cache": {
      "status": "up",
      "latency_ms": 2
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
psql -h localhost -p 5432 -U echo -d echo_db

# Via Docker
docker exec -it echo-postgres psql -U echo -d echo_db
```

### Running Queries

```sql
-- View all schemas
\dn

-- Switch to auth schema
SET search_path TO auth;

-- List tables in current schema
\dt

-- List all tables across schemas
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY schemaname, tablename;

-- View users (EMAIL-BASED)
SELECT id, email, phone_number, email_verified, created_at
FROM auth.users
WHERE deleted_at IS NULL;

-- View active sessions
SELECT user_id, device_id, ip_address, created_at, expires_at
FROM auth.sessions
WHERE expires_at > NOW()
ORDER BY created_at DESC;

-- View login history
SELECT user_id, ip_address, success, created_at
FROM auth.login_history
ORDER BY created_at DESC
LIMIT 10;

-- View recent messages
SELECT id, conversation_id, sender_user_id, content, message_type, created_at
FROM messages.messages
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 10;

-- View conversations
SELECT id, conversation_type, created_at
FROM messages.conversations
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- View user profiles
SELECT id, user_id, username, display_name, bio, created_at
FROM users.profiles
WHERE deleted_at IS NULL;
```

### Schema Information

```sql
-- View auth schema tables
\dt auth.*

-- View users schema tables
\dt users.*

-- View messages schema tables
\dt messages.*

-- Describe specific table
\d auth.users
\d messages.messages
\d users.profiles
```

## API Testing

### Using cURL

**Register User** (EMAIL-BASED):
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "phone_number": "+1234567890",
    "accept_terms": true
  }' | jq '.'

# Response:
# {
#   "success": true,
#   "data": {
#     "user_id": "550e8400-e29b-41d4-a716-446655440000",
#     "email": "user@example.com",
#     "email_verified": false,
#     "created_at": "2025-01-15T10:30:00Z"
#   }
# }
```

**Login**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "device_id": "device-12345",
    "device_name": "MacBook Pro"
  }' | jq '.'

# Response:
# {
#   "success": true,
#   "data": {
#     "user_id": "550e8400-e29b-41d4-a716-446655440000",
#     "session_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
#     "access_token": "eyJhbGciOiJIUzI1NiIs...",
#     "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
#     "expires_at": "2025-01-15T11:30:00Z"
#   }
# }
```

**Store Token**:
```bash
# Extract access token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "SecurePass123!"}' \
  | jq -r '.data.access_token')

echo "Token: $TOKEN"
```

**Create User Profile**:
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "display_name": "John Doe",
    "bio": "Software engineer"
  }' | jq '.'
```

**Get User Profile**:
```bash
USER_ID="550e8400-e29b-41d4-a716-446655440000"
curl -X GET "http://localhost:8080/api/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.'
```

**Send Message**:
```bash
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
    "content": "Hello, how are you?",
    "message_type": "text"
  }' | jq '.'
```

**Get Messages**:
```bash
CONVERSATION_ID="660e8400-e29b-41d4-a716-446655440001"
curl -X GET "http://localhost:8080/api/v1/messages?conversation_id=$CONVERSATION_ID&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.'
```

**Create Conversation**:
```bash
curl -X POST http://localhost:8080/api/v1/messages/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_type": "direct",
    "participant_ids": ["user-id-1", "user-id-2"]
  }' | jq '.'
```

### Using HTTPie

```bash
# Install HTTPie
brew install httpie  # macOS
apt install httpie   # Linux

# Register (email-based)
http POST localhost:8080/api/v1/auth/register \
  email="user@example.com" \
  password="SecurePass123!" \
  accept_terms:=true

# Login
http POST localhost:8080/api/v1/auth/login \
  email="user@example.com" \
  password="SecurePass123!"

# Save token
TOKEN=$(http POST localhost:8080/api/v1/auth/login \
  email="user@example.com" password="SecurePass123!" \
  | jq -r '.data.access_token')

# Authenticated request
http GET localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  Authorization:"Bearer $TOKEN"
```

## WebSocket Testing

### Using wscat

```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c "ws://localhost:8083/ws" \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Device-ID: device-12345" \
  -H "X-Platform: web"

# You'll receive connection acknowledgment:
# < {"type":"connection_ack","payload":{"status":"connected","timestamp":"2025-01-15T10:30:00Z","client_id":"conn_abc123"}}

# Send ping (keep-alive)
> {"type":"ping","payload":{}}
# < {"type":"pong","payload":{"timestamp":"2025-01-15T10:30:00Z"}}

# Send typing indicator
> {"type":"typing","payload":{"conversation_id":"660e8400-e29b-41d4-a716-446655440001","is_typing":true}}

# Send read receipt
> {"type":"read_receipt","payload":{"message_id":"770e8400-e29b-41d4-a716-446655440002"}}

# Receive messages (broadcasted by server)
# < {"type":"message","payload":{"id":"770e8400...","conversation_id":"660e8400...","sender_user_id":"550e8400...","content":"Hello!","message_type":"text","created_at":"2025-01-15T10:30:00Z"}}
```

### Using Browser Console

```javascript
// Connect to WebSocket
const userId = '550e8400-e29b-41d4-a716-446655440000';
const deviceId = 'device-12345';
const platform = 'web';

const ws = new WebSocket('ws://localhost:8083/ws');

// Note: Browser WebSocket doesn't support custom headers
// Server extracts user_id from JWT token or context
// For testing, you may need to modify server to accept query params

// Handle connection open
ws.onopen = () => {
  console.log('WebSocket Connected');

  // Server automatically sends connection_ack
};

// Handle incoming messages
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);

  switch(message.type) {
    case 'connection_ack':
      console.log('Connection established:', message.payload);

      // Send ping to keep connection alive
      ws.send(JSON.stringify({
        type: 'ping',
        payload: {}
      }));
      break;

    case 'pong':
      console.log('Pong received');
      break;

    case 'message':
      console.log('New message:', message.payload);
      break;

    case 'error':
      console.error('Error:', message.payload);
      break;
  }
};

// Send typing indicator
function sendTyping(conversationId, isTyping) {
  ws.send(JSON.stringify({
    type: 'typing',
    payload: {
      conversation_id: conversationId,
      is_typing: isTyping
    }
  }));
}

// Send read receipt
function sendReadReceipt(messageId) {
  ws.send(JSON.stringify({
    type: 'read_receipt',
    payload: {
      message_id: messageId
    }
  }));
}

// Heartbeat (send ping every 30 seconds)
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'ping',
      payload: {}
    }));
  }
}, 30000);

// Handle errors
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

// Handle close
ws.onclose = () => {
  console.log('WebSocket disconnected');
  // Implement reconnection logic with exponential backoff
};
```

### Complete WebSocket Client Example

See [WEBSOCKET_PROTOCOL.md](./WEBSOCKET_PROTOCOL.md#client-implementation) for full JavaScript client implementation with reconnection logic.

### Supported WebSocket Events

**Client → Server:**
1. `ping` - Keep-alive heartbeat
2. `typing` - Typing indicator
3. `read_receipt` - Mark message as read

**Server → Client:**
1. `connection_ack` - Connection established
2. `pong` - Heartbeat response
3. `message` - New message broadcast
4. `error` - Error occurred

## Debugging

### Viewing Logs

**Console Logs** (development):
```bash
docker logs echo-auth-service --tail 100 -f
```

**JSON Logs** (production):
```bash
docker logs echo-auth-service --tail 100 -f | jq '.'
```

**Filter by Log Level**:
```bash
# Errors only (console format)
docker logs echo-auth-service 2>&1 | grep "ERROR"

# JSON format
docker logs echo-auth-service 2>&1 | jq 'select(.level == "error")'
```

**Filter by Request ID**:
```bash
# Console format
docker logs echo-auth-service 2>&1 | grep "request_id=req_abc123"

# JSON format
docker logs echo-auth-service 2>&1 | jq 'select(.request_id == "req_abc123")'
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

**View Active Connections**:
```sql
SELECT pid, usename, application_name, client_addr, state, query
FROM pg_stat_activity
WHERE datname = 'echo_db';
```

**Check Table Sizes**:
```sql
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

**View Indexes**:
```sql
SELECT
  schemaname,
  tablename,
  indexname,
  indexdef
FROM pg_indexes
WHERE schemaname = 'auth'
ORDER BY tablename, indexname;
```

### Redis Debugging

```bash
# Connect to Redis
make redis-connect

# Or directly
docker exec -it echo-redis redis-cli -a redis_password

# Monitor all commands
MONITOR

# View all keys
KEYS "*"

# View session keys
KEYS "session:*"

# Get key value
GET "session:7c9e6679-7425-40de-944b-e07fc1f90ae7"

# View key TTL
TTL "session:7c9e6679-7425-40de-944b-e07fc1f90ae7"

# Get key type
TYPE "session:abc123"
```

### Kafka Debugging

```bash
# List topics
docker exec -it echo-kafka kafka-topics \
  --bootstrap-server localhost:9092 \
  --list

# Consume messages from notifications topic
docker exec -it echo-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic notifications \
  --from-beginning

# View consumer groups
docker exec -it echo-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --list
```

## Common Workflows

### Complete Registration and Login Flow

```bash
# 1. Register new user (email-based)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePass123!",
    "accept_terms": true
  }' | jq '.'

# 2. Login to get access token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePass123!",
    "device_id": "my-device",
    "device_name": "My Computer"
  }' | jq -r '.data.access_token')

echo "Access Token: $TOKEN"

# 3. Create user profile
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "display_name": "New User",
    "bio": "Just joined!"
  }' | jq '.'
```

### Sending and Receiving Messages

```bash
# 1. Create a conversation
CONVERSATION=$(curl -s -X POST http://localhost:8080/api/v1/messages/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_type": "direct",
    "participant_ids": ["user-id-1", "user-id-2"]
  }' | jq -r '.data.id')

echo "Conversation ID: $CONVERSATION"

# 2. Send a message
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"conversation_id\": \"$CONVERSATION\",
    \"content\": \"Hello from the API!\",
    \"message_type\": \"text\"
  }" | jq '.'

# 3. Get messages from conversation
curl -X GET "http://localhost:8080/api/v1/messages?conversation_id=$CONVERSATION&limit=50" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.'
```

### Adding a New Endpoint

See [SERVER_ARCHITECTURE.md](./SERVER_ARCHITECTURE.md#creating-a-new-service) for complete guide.

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

# Or specific service
docker logs echo-auth-service
docker logs echo-message-service
docker logs echo-postgres
```

**Common Issues**:
- Port already in use: `lsof -i :8080` and kill process
- Docker daemon not running: `docker info`
- Missing environment variables: check `.env` file

**Restart Services**:
```bash
make down
make up
```

### Database Connection Errors

**Test Connection**:
```bash
psql -h localhost -p 5432 -U echo -d echo_db -c "SELECT 1"
```

**Check PostgreSQL Status**:
```bash
docker ps | grep postgres
docker logs echo-postgres
```

**Check if database exists**:
```bash
docker exec -it echo-postgres psql -U echo -l
```

### Redis Connection Errors

**Test Connection**:
```bash
docker exec -it echo-redis redis-cli -a redis_password ping
# Should return: PONG
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
docker ps | grep zookeeper
docker logs echo-kafka
docker logs echo-zookeeper
```

**Test Kafka**:
```bash
docker exec -it echo-kafka kafka-topics \
  --bootstrap-server localhost:9092 \
  --list
```

### WebSocket Connection Refused

**Check Message Service**:
```bash
docker ps | grep message-service
docker logs echo-message-service
```

**Test WebSocket Endpoint**:
```bash
# Check if service is listening
curl -i http://localhost:8083/health
```

**Common Issues**:
- Missing headers (X-User-ID required)
- Service not running
- Port 8083 not exposed

### High Memory Usage

**Check Container Stats**:
```bash
docker stats
```

**Optimize Connection Pool** (`services/auth-service/configs/config.yaml`):
```yaml
database:
  postgres:
    max_open_conns: 10  # Reduce from 25
    max_idle_conns: 5   # Reduce from 10
```

**Restart Service**:
```bash
make auth-rerun
```

### Slow Response Times

**Check Request Duration** (in logs):
```bash
docker logs echo-auth-service 2>&1 | grep "duration"
```

**Add Database Indexes**:
```sql
CREATE INDEX idx_users_email ON auth.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_conversation ON messages.messages(conversation_id, created_at DESC);
CREATE INDEX idx_sessions_user ON auth.sessions(user_id, expires_at DESC);
```

---

For more information:
- [Architecture Documentation](./ARCHITECTURE.md)
- [Server Architecture](./SERVER_ARCHITECTURE.md)
- [API Reference](./API_REFERENCE.md)
- [WebSocket Protocol](./WEBSOCKET_PROTOCOL.md)
- [Database Schema](./DATABASE_SCHEMA.md)

**Last Updated**: January 2025
**Based On**: Actual implementation
