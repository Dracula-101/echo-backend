# Makefile Commands - Message Service

## Overview

The Makefile has been updated with comprehensive commands for the Message Service and Kafka. All commands follow the same pattern as other services (auth, gateway, location).

## ✅ What Was Added

### 1. **Message Service Commands** (7 commands)
- `make message-up` - Start Message Service
- `make message-down` - Stop Message Service
- `make message-restart` - Restart Message Service
- `make message-rerun` - Stop, rebuild, and start
- `make message-build` - Rebuild Docker image
- `make message-rebuild` - Rebuild with no cache (alias)
- `make message-logs` - View live logs

### 2. **Kafka Commands** (5 commands)
- `make kafka-up` - Start Kafka & Zookeeper
- `make kafka-down` - Stop Kafka & Zookeeper
- `make kafka-restart` - Restart Kafka
- `make kafka-logs` - View Kafka logs
- `make kafka-topics` - List all Kafka topics
- `make kafka-create-topics` - Create required topics (messages, notifications)

### 3. **Updated Main Commands**
- `make up` - Now includes Message Service + Kafka
- `make status` - Now includes Message Service + Kafka status
- `make health` - Now includes Message Service + Kafka health checks
- `make help` - Updated with Message Service + Kafka commands

---

## 🚀 Quick Start

### Start Everything (Recommended)

```bash
# Start all services including message service and Kafka
make up

# View status
make status

# Check health
make health
```

### Start Only Message Service

```bash
# This will also start dependencies (Postgres, Redis, Kafka, Zookeeper)
make message-up
```

### View Logs

```bash
# Follow message service logs
make message-logs

# Follow Kafka logs
make kafka-logs

# Follow all logs
make logs
```

---

## 📋 Message Service Commands

### Start Message Service
```bash
make message-up
```
**Output:**
```
💬 Starting Message Service...
✓ Message Service started
REST API:   http://localhost:8083
WebSocket:  ws://localhost:8083/ws
```

### Stop Message Service
```bash
make message-down
```

### Restart Message Service
```bash
make message-restart
```
Use this when you want to restart without rebuilding (faster).

### Rerun Message Service
```bash
make message-rerun
```
Use this to stop and start fresh (useful after code changes).

### Rebuild Message Service
```bash
make message-build
```
Rebuilds the Docker image with no cache. Use after:
- Changing Dockerfile.dev
- Adding new Go dependencies
- Major code refactoring

### View Logs
```bash
make message-logs
```
Shows live logs with automatic scrolling. Press `Ctrl+C` to exit.

---

## 📨 Kafka Commands

### Start Kafka
```bash
make kafka-up
```
**Output:**
```
📨 Starting Kafka & Zookeeper...
✓ Kafka started
Kafka: localhost:9093
```

### Stop Kafka
```bash
make kafka-down
```

### Restart Kafka
```bash
make kafka-restart
```

### List Topics
```bash
make kafka-topics
```
**Example Output:**
```
📋 Kafka Topics:
messages
notifications
__consumer_offsets
```

### Create Topics
```bash
make kafka-create-topics
```
Creates the required topics:
- `messages` (3 partitions)
- `notifications` (3 partitions)

### View Logs
```bash
make kafka-logs
```

---

## 🔍 Status & Health Checks

### Check Service Status
```bash
make status
```
**Output includes:**
```
📊 Service Status:

Message Service:
Running : ✅
Uptime  : 2024-11-04T10:30:00Z

Kafka:
Running : ✅
Uptime  : 2024-11-04T10:29:45Z

Zookeeper:
Running : ✅
Uptime  : 2024-11-04T10:29:30Z
```

### Check Health
```bash
make health
```
**Output includes:**
```
🏥 Checking service health...

Message Service:
✓ Healthy

Kafka:
✓ Ready
```

---

## 🛠️ Common Workflows

### 1. First Time Setup
```bash
# Start everything
make up

# Wait for services to be ready (~30 seconds)
make health

# Create Kafka topics
make kafka-create-topics

# View message service logs
make message-logs
```

### 2. Development Workflow
```bash
# Edit code in services/message-service/internal/
# Code changes are automatically detected by Air

# View logs to see rebuild
make message-logs

# If needed, restart manually
make message-restart
```

### 3. Troubleshooting
```bash
# Check if services are running
make status

# View message service logs
make message-logs

# Rebuild message service
make message-build

# Rerun message service
make message-rerun

# Check Kafka topics
make kafka-topics
```

### 4. Testing
```bash
# Start services
make up

# Test message service health
curl http://localhost:8083/health

# Send test message
curl -X POST http://localhost:8083/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $(uuidgen)" \
  -d '{
    "conversation_id": "'"$(uuidgen)"'",
    "content": "Test message",
    "message_type": "text"
  }'

# View logs to see the message processed
make message-logs
```

### 5. Clean Slate
```bash
# Stop everything and remove volumes
make clean

# Start fresh
make up
```

---

## 📊 Updated Help Output

Run `make help` to see all commands:

```bash
make help
```

**New sections in help:**

```
💬 Message Service:
  make message-up           - Start Message Service
  make message-down         - Stop Message Service
  make message-rerun        - Rerun Message Service
  make message-restart      - Restart Message Service
  make message-build        - Rebuild Message Service
  make message-logs         - View Message Service logs

📨 Kafka:
  make kafka-up             - Start Kafka & Zookeeper
  make kafka-down           - Stop Kafka & Zookeeper
  make kafka-logs           - View Kafka logs
  make kafka-restart        - Restart Kafka
  make kafka-topics         - List Kafka topics
```

---

## 🎯 Command Patterns

All service commands follow the same pattern:

| Command | Pattern | Example |
|---------|---------|---------|
| Start | `make [service]-up` | `make message-up` |
| Stop | `make [service]-down` | `make message-down` |
| Restart | `make [service]-restart` | `make message-restart` |
| Logs | `make [service]-logs` | `make message-logs` |
| Build | `make [service]-build` | `make message-build` |
| Rerun | `make [service]-rerun` | `make message-rerun` |

---

## 🔗 Service Dependencies

When you run `make message-up`, it automatically starts:
1. PostgreSQL (database)
2. Redis (cache)
3. Zookeeper (Kafka dependency)
4. Kafka (message broker)
5. Message Service

All dependencies are managed by Docker Compose's `depends_on` configuration.

---

## 💡 Tips

### Hot Reload
- Code changes in `services/message-service/` are automatically detected
- No need to restart manually
- Watch the rebuild in logs: `make message-logs`

### Multiple Services
```bash
# Start specific services
docker-compose -f infra/docker/docker-compose.dev.yml up -d postgres redis kafka message-service

# Or use the Makefile
make message-up  # Starts all dependencies automatically
```

### Debugging
```bash
# Follow logs for all services
make logs

# Follow logs for specific service
make message-logs    # Message Service
make kafka-logs      # Kafka
make auth-logs       # Auth Service
make gateway-logs    # API Gateway
```

### Performance
```bash
# Restart is faster than rerun
make message-restart  # ~2 seconds
make message-rerun    # ~10 seconds

# Use restart for quick changes
# Use rerun for major changes
```

---

## 🆘 Troubleshooting

### Message Service Won't Start

```bash
# Check logs
make message-logs

# Check dependencies
make status

# Rebuild
make message-build
make message-up
```

### Kafka Issues

```bash
# Check Kafka status
make status

# View Kafka logs
make kafka-logs

# Restart Kafka
make kafka-restart

# List topics
make kafka-topics
```

### Port Conflicts

If you see "port already in use" errors:

```bash
# Check what's using the port
lsof -i :8083  # Message Service
lsof -i :9092  # Kafka

# Stop conflicting processes or use docker compose
make down
make up
```

---

## 📝 Summary

**Message Service:**
- ✅ 7 commands for full lifecycle management
- ✅ Hot reload with Air
- ✅ Health checks integrated
- ✅ Same pattern as other services

**Kafka:**
- ✅ 5 commands for Kafka management
- ✅ Topic creation and listing
- ✅ Integrated with message service
- ✅ Health checks included

**Integration:**
- ✅ Updated `make up` to include all services
- ✅ Updated `make status` with new services
- ✅ Updated `make health` with new checks
- ✅ Updated `make help` with new commands

All commands are now ready to use! 🚀
