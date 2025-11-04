# Message Service - Docker Development Setup

## Overview

The message service is fully integrated into the Echo Backend development Docker environment with hot reload support via Air.

## What Was Added

### 1. **Dockerfile.dev**
- Based on `golang:1.25-alpine`
- Includes Air for hot reload
- Exposes port **8083** for HTTP/REST and WebSocket
- Mounts source code as volume for live reloading

### 2. **.air.toml**
- Configuration for Air hot reload
- Watches `internal/` and `cmd/` directories
- Automatically rebuilds on file changes
- Excludes test files and vendor directories

### 3. **.env.docker**
- Docker-specific environment configuration
- Pre-configured to work with Docker Compose services
- Database, Kafka, and Redis connection settings

### 4. **docker-compose.dev.yml Updates**
Added the following services:

#### **Zookeeper** (port 2181)
- Required dependency for Kafka
- Persistent data storage

#### **Kafka** (ports 9092, 9093)
- Message broker for async operations
- Auto-creates topics
- Exposed on port 9092 (internal) and 9093 (host)

#### **Message Service** (port 8083)
- Depends on: PostgreSQL, Redis, Kafka
- Hot reload enabled
- Health checks configured
- Persistent Go module cache

## Quick Start

### 1. Start All Services

From the project root:

```bash
cd /Users/pratik/Desktop/Projects/echo-backend
docker compose -f infra/docker/docker-compose.dev.yml up -d
```

### 2. View Message Service Logs

```bash
# Follow logs
docker logs -f echo-message-service

# View last 100 lines
docker logs --tail 100 echo-message-service
```

### 3. Start Only Message Service (with dependencies)

```bash
docker compose -f infra/docker/docker-compose.dev.yml up -d postgres redis kafka message-service
```

### 4. Rebuild Message Service

```bash
# Rebuild and restart
docker compose -f infra/docker/docker-compose.dev.yml up -d --build message-service
```

### 5. Stop Services

```bash
# Stop all
docker compose -f infra/docker/docker-compose.dev.yml down

# Stop and remove volumes (clean slate)
docker compose -f infra/docker/docker-compose.dev.yml down -v
```

## Service Dependencies

The message service depends on:

```
message-service
├── postgres (healthy)
├── redis (healthy)
├── kafka (healthy)
│   └── zookeeper (healthy)
└── db-init (completed)
```

## Ports

| Service | Internal Port | Host Port | Purpose |
|---------|--------------|-----------|---------|
| Message Service | 8083 | 8083 | HTTP/REST + WebSocket |
| PostgreSQL | 5432 | 5432 | Database |
| Redis | 6379 | 6379 | Cache |
| Kafka | 9092 | 9092 | Message broker (internal) |
| Kafka | 9093 | 9093 | Message broker (host) |
| Zookeeper | 2181 | 2181 | Kafka coordination |

## Testing the Service

### 1. Check Health

```bash
curl http://localhost:8083/health
```

Expected response:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "service": "message-service"
  }
}
```

### 2. Send a Test Message

```bash
curl -X POST http://localhost:8083/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-User-ID: <your-user-uuid>" \
  -d '{
    "conversation_id": "<conversation-uuid>",
    "content": "Hello from Docker!",
    "message_type": "text"
  }'
```

### 3. Connect to WebSocket

```bash
# Using websocat (install: brew install websocat)
websocat -H="X-User-ID: <your-user-uuid>" ws://localhost:8083/ws
```

Or use JavaScript:
```javascript
const ws = new WebSocket('ws://localhost:8083/ws', {
  headers: {
    'X-User-ID': '<your-user-uuid>',
    'X-Device-ID': 'test-device'
  }
});

ws.onmessage = (event) => {
  console.log('Received:', JSON.parse(event.data));
};
```

## Hot Reload

The service uses Air for automatic hot reload. When you edit any Go files in:
- `services/message-service/internal/`
- `services/message-service/cmd/`
- `shared/pkg/`

The service will automatically rebuild and restart.

**Watch the logs to see rebuilds:**
```bash
docker logs -f echo-message-service
```

## Troubleshooting

### Service Won't Start

1. **Check dependencies are healthy:**
```bash
docker compose -f infra/docker/docker-compose.dev.yml ps
```

2. **Check message service logs:**
```bash
docker logs echo-message-service
```

3. **Rebuild from scratch:**
```bash
docker compose -f infra/docker/docker-compose.dev.yml down -v
docker compose -f infra/docker/docker-compose.dev.yml build --no-cache message-service
docker compose -f infra/docker/docker-compose.dev.yml up -d
```

### Kafka Connection Issues

1. **Verify Kafka is healthy:**
```bash
docker logs echo-kafka
```

2. **Test Kafka connectivity:**
```bash
docker exec -it echo-kafka kafka-topics --list --bootstrap-server localhost:9092
```

3. **Create topics manually if needed:**
```bash
docker exec -it echo-kafka kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic messages \
  --partitions 3 \
  --replication-factor 1
```

### Database Connection Issues

1. **Check PostgreSQL logs:**
```bash
docker logs echo-postgres
```

2. **Verify database exists:**
```bash
docker exec -it echo-postgres psql -U echo -d echo_db -c "SELECT version();"
```

3. **Check message schema:**
```bash
docker exec -it echo-postgres psql -U echo -d echo_db -c "\dn"
```

### Hot Reload Not Working

1. **Check Air logs:**
```bash
docker logs -f echo-message-service | grep -i "air"
```

2. **Verify volumes are mounted:**
```bash
docker inspect echo-message-service | grep -A 10 "Mounts"
```

3. **Force rebuild:**
```bash
docker compose -f infra/docker/docker-compose.dev.yml restart message-service
```

## Environment Variables

The service uses these key environment variables (pre-configured in docker-compose):

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8083

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=echo
DB_PASSWORD=echo_password
DB_NAME=echo_db

# Kafka
KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC=messages
KAFKA_NOTIFICATION_TOPIC=notifications

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=redis_password

# Logging
LOG_LEVEL=debug
LOG_FORMAT=console
```

## Accessing Services from Host

| Service | URL |
|---------|-----|
| Message Service | `http://localhost:8083` |
| WebSocket | `ws://localhost:8083/ws` |
| PostgreSQL | `postgresql://echo:echo_password@localhost:5432/echo_db` |
| Redis | `redis://:redis_password@localhost:6379/0` |
| Kafka (host) | `localhost:9093` |

## Docker Compose Commands Reference

```bash
# Start all services
docker compose -f infra/docker/docker-compose.dev.yml up -d

# Start specific service
docker compose -f infra/docker/docker-compose.dev.yml up -d message-service

# View logs
docker compose -f infra/docker/docker-compose.dev.yml logs -f message-service

# Stop services
docker compose -f infra/docker/docker-compose.dev.yml stop

# Remove services (keeps volumes)
docker compose -f infra/docker/docker-compose.dev.yml down

# Remove services and volumes (clean slate)
docker compose -f infra/docker/docker-compose.dev.yml down -v

# Rebuild service
docker compose -f infra/docker/docker-compose.dev.yml build message-service

# Restart service
docker compose -f infra/docker/docker-compose.dev.yml restart message-service

# View service status
docker compose -f infra/docker/docker-compose.dev.yml ps
```

## Production Deployment

For production deployment, create:
- `Dockerfile` (production build, no Air)
- `docker-compose.prod.yml` with production configurations
- Proper secrets management
- TLS/SSL certificates
- Load balancer configuration

See the main `README.md` for production deployment guidelines.

## Next Steps

1. ✅ Start the Docker environment
2. ✅ Verify all services are healthy
3. ✅ Test the health endpoint
4. ✅ Send test messages
5. ✅ Connect via WebSocket
6. ✅ Monitor logs during development

The Docker development environment is production-like and includes all necessary dependencies for the message service!
