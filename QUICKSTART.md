# Echo Backend - Quick Start Guide

![Echo Backend](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat&logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)

A production-ready microservices backend for a real-time messaging application built with Go, PostgreSQL, Redis, and gRPC.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Service Architecture](#service-architecture)
5. [Database Schema](#database-schema)
6. [API Gateway](#api-gateway)
7. [Development Workflow](#development-workflow)
8. [Testing](#testing)
9. [Production Deployment](#production-deployment)
10. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

Echo Backend is a **microservices-based messaging platform** with the following components:

```
┌─────────────────────────────────────────────────┐
│         Client Applications                     │
│    (Web, iOS, Android, Desktop)                 │
└────────────────┬────────────────────────────────┘
                 │ HTTP/REST
                 ▼
┌─────────────────────────────────────────────────┐
│         API Gateway (Port 8080)                 │
│  • Request routing & load balancing             │
│  • Rate limiting & authentication               │
│  • Request/response validation                  │
└────┬──────────────┬──────────────────┬──────────┘
     │              │                  │
     ▼              ▼                  ▼
┌──────────┐  ┌──────────────┐  ┌──────────────┐
│ Auth     │  │ Message      │  │ User         │
│ Service  │  │ Service      │  │ Service      │
│ (8081)   │  │ (gRPC)       │  │ (gRPC)       │
└────┬─────┘  └──────────────┘  └──────────────┘
     │
     ▼
┌──────────────────────────────────────────────────┐
│         Location Service (8090)                  │
│         IP Geolocation & Location Tracking       │
└──────────────────────────────────────────────────┘

         Shared Infrastructure:
┌──────────────────────────────────────────────────┐
│ PostgreSQL (5432) • Redis (6379)                 │
│ Kafka (planned) • Jaeger (tracing)               │
└──────────────────────────────────────────────────┘
```

### Microservices

| Service | Port | Protocol | Status | Description |
|---------|------|----------|--------|-------------|
| **API Gateway** | 8080 | HTTP/REST | ✅ Active | Entry point, routing, rate limiting |
| **Auth Service** | 8081 | HTTP/REST | ✅ Active | Authentication, JWT, sessions |
| **Location Service** | 8090 | HTTP/REST | ✅ Active | IP geolocation lookups |
| **Message Service** | 50051 | gRPC | 🚧 Stub | Core messaging functionality |
| **User Service** | 50052 | gRPC | 🚧 Stub | User profiles & contacts |
| **Notification Service** | - | gRPC | 🚧 Stub | Push notifications |
| **Media Service** | - | HTTP | 🚧 Stub | File uploads & media |
| **Presence Service** | - | gRPC | 🚧 Stub | Online status tracking |
| **Analytics Service** | - | gRPC | 🚧 Stub | User analytics & metrics |

---

## Prerequisites

### Required Software

- **Docker Desktop** 4.20+ ([Download](https://www.docker.com/products/docker-desktop))
- **Docker Compose** 2.x (included with Docker Desktop)
- **Go** 1.25+ ([Download](https://go.dev/dl/)) - only for local development
- **Make** (usually pre-installed on macOS/Linux)
- **PostgreSQL Client** (optional, for direct DB access)

### System Requirements

- **RAM:** 4GB minimum, 8GB recommended
- **Disk:** 5GB free space
- **OS:** macOS, Linux, or Windows with WSL2

### Verify Installation

```bash
docker --version          # Should show Docker version 20.10+
docker-compose --version  # Should show version 2.x
go version               # Should show go1.25 or higher
make --version           # Should show GNU Make
```

---

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/echo-backend.git
cd echo-backend
```

### 2. Environment Setup

Create environment configuration files:

```bash
# Create .env file in project root
touch .env

# Add the following configuration
cat > .env << 'EOF'
# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=echo
POSTGRES_PASSWORD=echo_password
POSTGRES_DB=echo_db

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis_password

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Application Environment
APP_ENV=development
LOG_LEVEL=debug
EOF
```

### 3. Start All Services

```bash
# Start all services (PostgreSQL, Redis, Auth, Gateway, Location)
make up

# Wait for services to initialize (~30 seconds)
# You should see output indicating services are starting
```

### 4. Initialize Database

```bash
# Initialize database schemas
make db-init

# Seed with test data (optional)
make db-seed
```

### 5. Verify Services

```bash
# Check health of all services
make health

# Expected output:
# ✓ API Gateway: Healthy
# ✓ Auth Service: Healthy
# ✓ PostgreSQL: Ready
# ✓ Redis: Ready
```

### 6. Test the API

```bash
# API Gateway Health Check
curl http://localhost:8080/health

# Expected response:
# {
#   "status": "ok",
#   "timestamp": "2024-11-03T..."
# }
```

### 🎉 You're Ready!

Your Echo Backend is now running:
- **API Gateway:** http://localhost:8080
- **PostgreSQL:** localhost:5432
- **Redis:** localhost:6379

---

## Service Architecture

### API Gateway (`services/api-gateway`)

**Purpose:** Single entry point for all client requests

**Features:**
- Reverse proxy with load balancing
- Rate limiting (fixed window, sliding window, token bucket)
- Request/response validation
- CORS handling
- Compression (gzip, deflate)
- Health monitoring

**Key Endpoints:**
```
GET  /health                     - Health check
GET  /metrics                    - Prometheus metrics
POST /api/v1/auth/register       - User registration
POST /api/v1/auth/login          - User login
POST /api/v1/auth/refresh        - Refresh token
GET  /api/v1/users/:id           - Get user profile
```

**Configuration:** `services/api-gateway/configs/config.yaml`

---

### Auth Service (`services/auth-service`)

**Purpose:** Handles authentication, authorization, and session management

**Features:**
- User registration with email/phone verification
- Password authentication (bcrypt hashing)
- JWT token generation (access + refresh tokens)
- Session management with Redis caching
- 2FA support (OTP via email/SMS)
- OAuth provider integration (Google, Facebook, Apple)
- Login history tracking
- Security event logging
- Device fingerprinting
- IP geolocation (via Location Service)

**Database Tables:**
- `auth.users` - User credentials
- `auth.sessions` - Active sessions
- `auth.otp_verifications` - OTP codes
- `auth.password_reset_tokens` - Password reset
- `auth.oauth_providers` - OAuth connections

**API Endpoints:**
```
POST /register              - Create new account
POST /login                 - Authenticate user
POST /refresh               - Refresh access token
POST /logout                - End session
POST /verify-email          - Verify email address
POST /verify-phone          - Verify phone number
POST /request-password-reset - Request password reset
POST /reset-password        - Reset password
GET  /sessions              - List active sessions
DELETE /sessions/:id        - Revoke session
```

---

### Location Service (`services/location-service`)

**Purpose:** IP geolocation and location tracking

**Features:**
- IP address to location mapping
- Country, region, city lookup
- Timezone detection
- ISP identification
- Latitude/longitude coordinates

**API Endpoints:**
```
GET /lookup?ip=<ip_address>  - Get location from IP
GET /health                  - Service health
```

**Database Tables:**
- `location.ip_blocks` - IP range mappings
- `location.locations` - Location data

---

### Message Service (`services/message-service`) 🚧

**Purpose:** Core messaging functionality (planned/stub)

**Planned Features:**
- Send/receive messages
- Real-time message delivery (WebSocket/gRPC streaming)
- Message encryption
- Read receipts
- Typing indicators
- File attachments
- Message reactions
- Message search
- Message history

**Database Tables:**
- `messages.conversations` - Chat threads
- `messages.messages` - Individual messages
- `messages.conversation_participants` - Conversation members
- `messages.reactions` - Message reactions
- `messages.delivery_status` - Delivery tracking
- `messages.calls` - Voice/video calls

---

### User Service (`services/user-service`) 🚧

**Purpose:** User profile and contact management (planned/stub)

**Planned Features:**
- User profile CRUD
- Avatar/cover image management
- Contact list management
- Friend requests
- User blocking
- User search
- Profile privacy settings

**Database Tables:**
- `users.profiles` - User profiles
- `users.contacts` - Friend connections
- `users.settings` - User preferences

---

### Presence Service (`services/presence-service`) 🚧

**Purpose:** Real-time presence tracking (planned/stub)

**Planned Features:**
- Online/offline status
- Last seen tracking
- Typing indicators
- Custom status messages
- Multi-device presence
- Presence subscriptions

**Database Tables:**
- `presence.user_status` - Current status
- `presence.connections` - Active connections
- `presence.typing_indicators` - Typing state
- `presence.activity_log` - Activity history

---

## Database Schema

### Schema Organization

Echo Backend uses **7 PostgreSQL schemas** for domain separation:

| Schema | Tables | Purpose |
|--------|--------|---------|
| `auth` | 6 | Authentication & sessions |
| `users` | 4 | User profiles & contacts |
| `messages` | 20 | Conversations & messaging |
| `media` | 2 | File storage & metadata |
| `notifications` | 2 | Push notifications |
| `analytics` | 6 | User analytics & events |
| `location` | 2 | IP geolocation data |
| `presence` | 6 | Real-time presence (new!) |

### Key Tables

#### Authentication (`auth` schema)

**auth.users** - User credentials and security
```sql
- id (UUID, PK)
- email (unique, indexed)
- phone_number (unique, optional)
- password_hash (bcrypt)
- two_factor_enabled
- account_status (active, suspended, banned)
- failed_login_attempts
- last_successful_login_at
```

**auth.sessions** - Active user sessions
```sql
- id (UUID, PK)
- user_id (FK to users)
- session_token (unique)
- refresh_token (unique)
- device_id, device_type, device_os
- ip_address, ip_country, ip_city
- fcm_token, apns_token (push notifications)
- expires_at
- last_activity_at
```

#### Messaging (`messages` schema)

**messages.conversations** - Chat threads
```sql
- id (UUID, PK)
- conversation_type (direct, group, channel, broadcast)
- title, description, avatar_url
- creator_user_id
- is_encrypted
- member_count, message_count
- last_message_at
```

**messages.messages** - Individual messages
```sql
- id (UUID, PK)
- conversation_id (FK)
- sender_user_id (FK)
- message_type (text, image, video, audio, etc.)
- content (encrypted if is_encrypted)
- status (sent, delivered, read, failed)
- is_edited, edit_history
- mentions, hashtags, links
- reply_count, reaction_count
```

#### User Profiles (`users` schema)

**users.profiles** - User profile information
```sql
- id (UUID, PK)
- user_id (FK to auth.users)
- username (unique)
- display_name
- bio, avatar_url, cover_image_url
- online_status (online, offline, away, busy)
- last_seen_at
- profile_visibility (public, friends, private)
```

#### Presence Tracking (`presence` schema) 🆕

**presence.user_status** - Real-time presence
```sql
- user_id (PK, FK to auth.users)
- status (online, offline, away, busy, invisible)
- last_seen_at, last_activity_at
- is_online, active_connections
- custom_status_text, custom_status_emoji
```

**presence.connections** - Active WebSocket connections
```sql
- id (UUID, PK)
- user_id (FK)
- connection_id (unique)
- device_id, device_type
- ip_address, user_agent
- connected_at, last_heartbeat_at
- server_id (for distributed systems)
```

### Database Functions & Triggers

**Automatic Timestamp Updates:**
```sql
CREATE TRIGGER update_updated_at
    BEFORE UPDATE ON <table>
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**Presence Helper Functions:**
```sql
-- Update user presence status
SELECT presence.update_user_status(user_id, 'online', device_id);

-- Clean up expired typing indicators
SELECT presence.cleanup_expired_typing();

-- Mark stale connections as disconnected
SELECT presence.cleanup_stale_connections();
```

---

## API Gateway

### Routing Configuration

Routes are defined in `services/api-gateway/configs/config.yaml`:

```yaml
services:
  - name: auth-service
    url: http://auth-service:8081
    health_check_path: /health
    timeout_seconds: 30
    routes:
      - path: /api/v1/auth
        strip_prefix: false
        rate_limit:
          requests_per_second: 10
          burst: 20

  - name: user-service
    url: http://user-service:8082
    routes:
      - path: /api/v1/users
        methods: [GET, PUT, PATCH, DELETE]
        require_auth: true
```

### Rate Limiting

Three strategies available:
1. **Fixed Window:** Simple request counting per time window
2. **Sliding Window:** More accurate, prevents burst at window edges
3. **Token Bucket:** Allows bursts with gradual refill

Configuration:
```yaml
rate_limiting:
  default_strategy: sliding_window
  default_limit: 100
  default_window: 60s

  # Per-endpoint overrides
  endpoints:
    - path: /api/v1/auth/login
      limit: 5
      window: 60s
      strategy: sliding_window
```

### Security Headers

Automatic security headers:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000`

### CORS Configuration

```yaml
cors:
  allowed_origins:
    - http://localhost:3000
    - https://app.example.com
  allowed_methods: [GET, POST, PUT, PATCH, DELETE, OPTIONS]
  allowed_headers: [Authorization, Content-Type]
  max_age: 86400
```

---

## Development Workflow

### Available Make Commands

```bash
# Service Management
make up                   # Start all services
make down                 # Stop all services
make restart              # Restart all services
make logs                 # View logs from all services
make status               # Show service status
make ps                   # List running containers

# Individual Services
make gateway-up           # Start API Gateway only
make auth-up              # Start Auth Service only
make location-up          # Start Location Service only

# Database Operations
make db-init              # Initialize database schemas
make db-seed              # Seed with test data
make db-connect           # Connect to PostgreSQL CLI
make db-reset             # Drop and recreate database

# Redis Operations
make redis-connect        # Connect to Redis CLI
make redis-flush          # Flush all Redis data

# Development
make health               # Check service health
make test                 # Run all tests
make setup                # Initial project setup
```

### Database Migrations

#### Using Migrations (Recommended)

```bash
# Run migrations
cd infra/scripts
chmod +x run-migrations.sh
./run-migrations.sh up

# Check migration status
./run-migrations.sh status

# Rollback last migration
./run-migrations.sh down

# Force re-apply specific migration
./run-migrations.sh force 2
```

#### Migration Files Structure

```
migrations/postgres/
├── 000001_init_schema.up.sql          # Initial baseline
├── 000001_init_schema.down.sql        # Rollback baseline
├── 000002_add_presence_schema.up.sql  # Add presence tracking
└── 000002_add_presence_schema.down.sql # Remove presence tracking
```

#### Creating New Migrations

```bash
# Create new migration files
cd migrations/postgres

# Example: Add index
cat > 000003_add_user_email_index.up.sql << 'EOF'
CREATE INDEX idx_users_email_lower ON auth.users(LOWER(email));

INSERT INTO schema_migrations (version, description)
VALUES (3, 'Add case-insensitive email index');
EOF

cat > 000003_add_user_email_index.down.sql << 'EOF'
DROP INDEX IF EXISTS idx_users_email_lower;
DELETE FROM schema_migrations WHERE version = 3;
EOF
```

### Test Accounts

After running `make db-seed`, these test accounts are available:

| Email | Username | Status | Password |
|-------|----------|--------|----------|
| alice@example.com | alice | Online | password123 |
| bob@example.com | bob | Online | password123 |
| charlie@example.com | charlie | Away | password123 |
| david@example.com | david | Offline | password123 |
| eve@example.com | eve | Busy | password123 |

### Hot Reload (Development Mode)

Services use [Air](https://github.com/cosmtrek/air) for hot reloading:

```bash
# Edit code in services/auth-service/
# Save file
# Air automatically rebuilds and restarts the service
```

Configuration: `.air.toml` in each service directory

---

## Testing

### Running Tests

```bash
# Run all tests
make test

# Test specific service
cd services/auth-service
go test -v ./...

# Test with coverage
go test -v -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### API Testing Examples

#### Register New User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePassword123!",
    "username": "newuser",
    "display_name": "New User"
  }'
```

#### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'

# Response:
# {
#   "access_token": "eyJhbGc...",
#   "refresh_token": "eyJhbGc...",
#   "user": { ... }
# }
```

#### Get User Profile (Authenticated)

```bash
# Use access_token from login response
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer eyJhbGc..."
```

---

## Production Deployment

### Environment Variables

Create `.env.production`:

```bash
# Database
POSTGRES_HOST=your-postgres-host.com
POSTGRES_PORT=5432
POSTGRES_USER=echo_prod
POSTGRES_PASSWORD=super-secure-password
POSTGRES_DB=echo_prod
DB_SSL_MODE=require
DB_MAX_CONNECTIONS=100

# Redis
REDIS_HOST=your-redis-host.com
REDIS_PORT=6379
REDIS_PASSWORD=super-secure-redis-password
REDIS_TLS=true

# JWT
JWT_SECRET=production-secret-key-min-32-chars-long
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Application
APP_ENV=production
LOG_LEVEL=info
LOG_FORMAT=json

# Observability
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
PROMETHEUS_PORT=9090
```

### Docker Production Build

```bash
# Build production images
docker-compose -f infra/docker/docker-compose.prod.yml build

# Start with production config
docker-compose -f infra/docker/docker-compose.prod.yml up -d

# Scale services
docker-compose -f infra/docker/docker-compose.prod.yml up -d --scale auth-service=3
```

### Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -f infra/kubernetes/namespace.yaml
kubectl apply -f infra/kubernetes/configmaps/
kubectl apply -f infra/kubernetes/secrets/
kubectl apply -f infra/kubernetes/deployments/
kubectl apply -f infra/kubernetes/services/
kubectl apply -f infra/kubernetes/ingress.yaml

# Check deployment status
kubectl get pods -n echo-backend
kubectl get svc -n echo-backend
```

### Health Checks

All services expose health endpoints:
```
GET /health

Response:
{
  "status": "ok",
  "timestamp": "2024-11-03T12:00:00Z",
  "dependencies": {
    "database": "connected",
    "redis": "connected",
    "location_service": "healthy"
  }
}
```

### Monitoring

- **Metrics:** Prometheus metrics at `/metrics`
- **Tracing:** OpenTelemetry + Jaeger
- **Logging:** Structured JSON logs (Zap)

---

## Troubleshooting

### Common Issues

#### 1. Services won't start

```bash
# Check Docker is running
docker ps

# Check logs
make logs

# Restart services
make down
make up
```

#### 2. Database connection errors

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
docker exec -it echo-postgres psql -U echo -d echo_db

# Reinitialize database
make db-reset
```

#### 3. Port already in use

```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>

# Or change port in docker-compose.dev.yml
```

#### 4. Redis connection refused

```bash
# Check Redis is running
docker ps | grep redis

# Test connection
docker exec -it echo-redis redis-cli -a redis_password PING

# Should respond: PONG
```

#### 5. Services can't communicate

```bash
# Check Docker network
docker network ls
docker network inspect echo-network

# Recreate network
make down
make clean
make up
```

### Debugging Tips

```bash
# View service logs
docker logs echo-auth-service
docker logs -f echo-api-gateway  # Follow mode

# Execute commands in container
docker exec -it echo-auth-service /bin/sh

# Check service health
curl http://localhost:8080/health
curl http://localhost:8081/health

# View database tables
make db-connect
\dt auth.*
\dt messages.*
```

### Getting Help

- **Documentation:** Check individual service README files
- **Issues:** Report at https://github.com/your-org/echo-backend/issues
- **Logs:** Always include relevant logs when reporting issues

---

## Next Steps

1. **Implement Message Service:**
   - Set up gRPC server
   - Implement message CRUD operations
   - Add real-time delivery (WebSocket/gRPC streaming)

2. **Complete User Service:**
   - User profile management
   - Contact operations
   - User search functionality

3. **Add Presence Service:**
   - WebSocket connections
   - Real-time status updates
   - Use new `presence` schema

4. **Notification Service:**
   - FCM/APNS integration
   - Push notification delivery
   - In-app notifications

5. **Media Service:**
   - File upload to cloud storage (S3/R2)
   - Image processing
   - Video transcoding

---

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [gRPC Go Tutorial](https://grpc.io/docs/languages/go/)
- [Docker Documentation](https://docs.docker.com/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)

---

**Happy Coding! 🚀**

For questions or contributions, please open an issue or submit a pull request.
