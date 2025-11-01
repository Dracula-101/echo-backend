# Docker Compose Configuration

## Overview

This directory contains the auth service Docker Compose configuration that sets up a minimal environment for running the auth service locally.

## Files

- **`docker-compose.auth.yml`** - Auth service with PostgreSQL and Redis only (minimal setup)
- **`../../infra/docker/docker-compose.yml`** - Full infrastructure setup (all services)

## Quick Start

### Auth Service Only (Minimal)

Use this for auth service development:

```bash
# From project root
docker-compose -f services/auth-service/docker-compose.auth.yml up -d

# Or using the helper script
./scripts/start-auth.sh

# Or using Make
make auth-up
```

**Services included:**
- PostgreSQL (database)
- Redis (cache)
- Auth Service

### Full Infrastructure

Use this when you need all backend services:

```bash
# From project root
docker-compose -f infra/docker/docker-compose.yml up -d
```

**Services included:**
- PostgreSQL
- Redis
- Kafka + Zookeeper
- TimescaleDB (analytics)
- Meilisearch (search)
- Prometheus (metrics)
- Grafana (monitoring)
- Jaeger (tracing)

## Service Configuration

### PostgreSQL
- **Port:** 5432
- **Database:** echo_db
- **User:** echo
- **Password:** echo_password (configurable via .env)
- **Auto-initialization:** Loads schemas, functions, and triggers from `database/` directory

### Redis
- **Port:** 6379
- **Password:** redis_password (configurable via .env)
- **Persistence:** Enabled with appendonly mode

### Auth Service

#### Production Mode
- **HTTP:** http://localhost:8080
- **gRPC:** localhost:50051
- Built from `Dockerfile`

#### Development Mode
- **HTTP:** http://localhost:8081
- **gRPC:** localhost:50052
- **Debugger:** localhost:2345
- Built from `Dockerfile.dev` with hot-reload

```bash
# Start with development profile
docker-compose -f services/auth-service/docker-compose.auth.yml --profile dev up -d
```

## Environment Variables

Create a `.env` file in the project root:

```env
# Database
POSTGRES_USER=echo
POSTGRES_PASSWORD=echo_password
POSTGRES_DB=echo_db

# Redis
REDIS_PASSWORD=redis_password

# JWT
JWT_SECRET=your-secret-key

# Application
APP_ENV=development
LOG_LEVEL=debug
```

## Network

All services run on the `echo-network` bridge network, allowing:
- Service discovery by container name
- Isolated networking
- Easy inter-service communication

## Volumes

- `postgres_data` - PostgreSQL data persistence
- `redis_data` - Redis data persistence
- `auth_logs` - Auth service logs
- `auth_keys` - Auth service keys (JWT, etc.)
- `auth_dev_cache` - Go module cache (development)

## Comparison

| Feature | auth-service compose | Full infrastructure |
|---------|---------------------|---------------------|
| PostgreSQL | ✅ | ✅ |
| Redis | ✅ | ✅ |
| Kafka | ❌ | ✅ |
| TimescaleDB | ❌ | ✅ |
| Meilisearch | ❌ | ✅ |
| Monitoring | ❌ | ✅ (Prometheus + Grafana) |
| Tracing | ❌ | ✅ (Jaeger) |
| Auth Service | ✅ | ❌ |

## When to Use Which

### Use `docker-compose.auth.yml` when:
- Developing the auth service
- Testing auth features
- You only need auth functionality
- Quick local development
- CI/CD pipelines for auth service

### Use `infra/docker/docker-compose.yml` when:
- Running the full backend
- Testing inter-service communication
- Need message queuing (Kafka)
- Need full-text search (Meilisearch)
- Need monitoring and tracing
- Production-like environment

## Combining Both

You can run both if needed (they share the same network):

```bash
# Start infrastructure services
docker-compose -f infra/docker/docker-compose.yml up -d postgres redis kafka

# Start auth service
docker-compose -f services/auth-service/docker-compose.auth.yml up -d auth-service
```

**Note:** Make sure not to start duplicate services (postgres, redis) in both files.

## Troubleshooting

### Port Conflicts
If ports are already in use:
1. Stop conflicting services
2. Or modify ports in the compose file:
   ```yaml
   ports:
     - "5433:5432"  # Use different external port
   ```

### Network Issues
If services can't communicate:
```bash
# Ensure network exists
docker network inspect echo-network

# Recreate if needed
docker network create echo-network
```

### Volume Issues
To reset data:
```bash
# WARNING: This deletes all data
docker-compose -f services/auth-service/docker-compose.auth.yml down -v
```

## See Also

- Main documentation: `../../README.md`
- Quick start: `../../START_HERE.md`
- Full guide: `../../QUICKSTART.md`
- Auth Docker guide: `../../README.auth-docker.md`
