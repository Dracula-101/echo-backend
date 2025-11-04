# Docker Development Setup - Summary

## Files Created ✅

### 1. **Dockerfile.dev** (1.2 KB)
**Location:** `/services/message-service/Dockerfile.dev`

**Purpose:** Development Docker image with hot reload support

**Features:**
- Based on `golang:1.25-alpine`
- Installs Air for live reloading
- Mounts source code as volumes
- Exposes port 8083
- Matches pattern from auth-service and api-gateway

**Key Configuration:**
```dockerfile
FROM golang:1.25-alpine
RUN go install github.com/air-verse/air@v1.52.3
EXPOSE 8083
CMD ["air", "-c", ".air.toml"]
```

---

### 2. **.air.toml** (1.0 KB)
**Location:** `/services/message-service/.air.toml`

**Purpose:** Air hot reload configuration

**Features:**
- Watches `internal/`, `cmd/`, and `shared/` directories
- Excludes test files and vendor directories
- Automatic rebuild on Go file changes
- 1-second delay between rebuilds
- Pre-commands to update module references

**Watched Extensions:**
- `.go`, `.yaml`, `.yml`, `.html`, `.tpl`, `.tmpl`

---

### 3. **.env.docker** (1.2 KB)
**Location:** `/services/message-service/.env.docker`

**Purpose:** Docker-specific environment variables

**Configuration:**
- Server: Port 8083, host 0.0.0.0
- Database: postgres:5432 (echo_db)
- Kafka: kafka:9092
- Redis: redis:6379
- WebSocket: Optimized settings
- Logging: Debug level, console format

---

### 4. **docker-compose.dev.yml** (Updated)
**Location:** `/infra/docker/docker-compose.dev.yml`

**Added Services:**

#### **Zookeeper** (Required for Kafka)
```yaml
zookeeper:
  image: confluentinc/cp-zookeeper:7.5.0
  ports: 2181
  volumes: zookeeper_data, zookeeper_logs
  healthcheck: enabled
```

#### **Kafka** (Message Broker)
```yaml
kafka:
  image: confluentinc/cp-kafka:7.5.0
  ports: 9092 (internal), 9093 (host)
  depends_on: zookeeper
  volumes: kafka_data
  healthcheck: enabled
  auto_create_topics: true
```

#### **Message Service**
```yaml
message-service:
  build: Dockerfile.dev
  ports: 8083
  depends_on:
    - postgres (healthy)
    - redis (healthy)
    - kafka (healthy)
    - db-init (completed)
  volumes:
    - source code (hot reload)
    - go module cache
    - build cache
  healthcheck: enabled
```

**Added Volumes:**
- `zookeeper_data`
- `zookeeper_logs`
- `kafka_data`
- `message-service-go-cache`
- `message-service-build-cache`

---

### 5. **DOCKER_SETUP.md** (Created)
**Location:** `/services/message-service/DOCKER_SETUP.md`

**Purpose:** Complete Docker development guide

**Contents:**
- Quick start guide
- Service dependencies diagram
- Port mappings
- Testing instructions
- Troubleshooting guide
- Docker commands reference

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                   Docker Compose Dev                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  Zookeeper   │  │    Kafka     │  │  PostgreSQL  │ │
│  │   :2181      │──│   :9092/93   │  │    :5432     │ │
│  └──────────────┘  └──────┬───────┘  └──────┬───────┘ │
│                           │                  │          │
│                    ┌──────▼──────────────────▼──────┐  │
│                    │    Message Service             │  │
│                    │       :8083                     │  │
│                    │  • REST API                     │  │
│                    │  • WebSocket                    │  │
│                    │  • Hot Reload (Air)             │  │
│                    └─────────────────────────────────┘  │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │    Redis     │  │ Auth Service │  │ API Gateway  │ │
│  │    :6379     │  │    :8081     │  │    :8080     │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Service Startup Order

```
1. PostgreSQL + Redis + Zookeeper (parallel)
   ↓
2. db-init (database schema initialization)
   ↓
3. Kafka (depends on Zookeeper)
   ↓
4. Auth Service, Location Service (parallel)
   ↓
5. API Gateway (depends on Auth Service)
   ↓
6. Message Service (depends on Postgres, Redis, Kafka, db-init)
```

## Quick Commands

### Start Everything
```bash
cd /Users/pratik/Desktop/Projects/echo-backend
docker compose -f infra/docker/docker-compose.dev.yml up -d
```

### View Message Service Logs
```bash
docker logs -f echo-message-service
```

### Rebuild Message Service
```bash
docker compose -f infra/docker/docker-compose.dev.yml up -d --build message-service
```

### Stop Everything
```bash
docker compose -f infra/docker/docker-compose.dev.yml down
```

### Clean Slate (Remove Volumes)
```bash
docker compose -f infra/docker/docker-compose.dev.yml down -v
```

## Port Mappings

| Service | Port | URL |
|---------|------|-----|
| Message Service | 8083 | http://localhost:8083 |
| WebSocket | 8083 | ws://localhost:8083/ws |
| Kafka (internal) | 9092 | kafka:9092 |
| Kafka (host) | 9093 | localhost:9093 |
| Zookeeper | 2181 | localhost:2181 |
| PostgreSQL | 5432 | localhost:5432 |
| Redis | 6379 | localhost:6379 |
| Auth Service | 8081 | http://localhost:8081 |
| API Gateway | 8080 | http://localhost:8080 |
| Location Service | 8090 | http://localhost:8090 |

## Testing the Setup

### 1. Check Health
```bash
curl http://localhost:8083/health
# Expected: {"success":true,"data":{"status":"healthy","service":"message-service"}}
```

### 2. Check All Services
```bash
docker compose -f infra/docker/docker-compose.dev.yml ps
# All services should be "healthy" or "running"
```

### 3. View Service Logs
```bash
# Message service
docker logs echo-message-service

# Kafka
docker logs echo-kafka

# PostgreSQL
docker logs echo-postgres
```

### 4. Send Test Message
```bash
curl -X POST http://localhost:8083/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $(uuidgen)" \
  -d '{
    "conversation_id": "'"$(uuidgen)"'",
    "content": "Test message from Docker!",
    "message_type": "text"
  }'
```

## Hot Reload

The service automatically reloads when you edit:
- `services/message-service/internal/**/*.go`
- `services/message-service/cmd/**/*.go`
- `shared/pkg/**/*.go`

Watch the rebuild in logs:
```bash
docker logs -f echo-message-service | grep -i "building\|running"
```

## Key Features

✅ **Production-like Environment**
- All services containerized
- Proper service dependencies
- Health checks enabled
- Persistent data volumes

✅ **Developer-Friendly**
- Hot reload with Air
- Source code mounted as volumes
- Debug logging enabled
- No need to rebuild for code changes

✅ **Scalable Design**
- Kafka for async messaging
- Redis for caching
- PostgreSQL for persistence
- WebSocket for real-time

✅ **Easy Troubleshooting**
- Comprehensive logging
- Health check endpoints
- Service status monitoring
- Clean restart capabilities

## File Pattern Consistency

All files follow the exact same pattern as existing services:

| File | Pattern Source |
|------|----------------|
| Dockerfile.dev | auth-service/Dockerfile.dev |
| .air.toml | auth-service/.air.toml |
| .env.docker | auth-service/.env.docker |
| docker-compose.dev.yml | Existing structure |

## What Makes This Production-Ready

1. **Service Discovery**: Services communicate using container names
2. **Health Checks**: All critical services have health checks
3. **Dependency Management**: Proper `depends_on` with conditions
4. **Persistent Storage**: Named volumes for data persistence
5. **Graceful Shutdown**: All services support SIGTERM/SIGINT
6. **Resource Limits**: Can be added in docker-compose (optional)
7. **Logging**: Structured logging to stdout/stderr
8. **Environment Configuration**: Externalized via environment variables

## Next Steps

1. ✅ Files created successfully
2. ✅ Docker configuration validated
3. ✅ Service dependencies configured
4. ✅ Hot reload enabled

**Ready to use!** Start the services with:
```bash
cd /Users/pratik/Desktop/Projects/echo-backend
docker compose -f infra/docker/docker-compose.dev.yml up -d
```

All services are now ready for development with hot reload! 🚀
