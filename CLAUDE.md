# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Echo Backend is a production-ready microservices architecture for a real-time messaging platform, built with Go. The system follows WhatsApp/Telegram-inspired design patterns with focus on scalability, observability, and clean architecture.

## Essential Commands

### Development Workflow
```bash
make up              # Start all services
make down            # Stop all services
make logs            # View logs from all services
make status          # Check service health
make health          # Run health checks
make rerun           # Stop, rebuild, and restart all services
```

### Individual Service Management
```bash
make <service>-up         # Start specific service (auth, gateway, message, location)
make <service>-down       # Stop specific service
make <service>-rerun      # Stop and restart service
make <service>-logs       # View service logs
make <service>-rebuild    # Rebuild service (no cache)
```

### Database Operations
```bash
make db-init             # Initialize database schemas
make db-migrate          # Run database migrations
make db-migrate-down     # Rollback last migration
make db-migrate-status   # Show migration status
make db-connect          # Connect to PostgreSQL CLI
make db-seed             # Seed with test data
make db-reset            # Drop and recreate database (destructive!)
```

### Infrastructure
```bash
make kafka-topics        # List Kafka topics
make kafka-create-topics # Create required topics
make redis-connect       # Connect to Redis CLI
```

### Testing
```bash
make test               # Run tests across all services
make test-auth          # Test auth endpoints
make verify-security    # Verify auth service isolation
```

### Building Individual Services
```bash
cd services/<service-name>
go build -o bin/<service-name> cmd/server/main.go
go run cmd/server/main.go  # Development mode
```

## Architecture

### Monorepo Structure

```
echo-backend/
├── services/           # Microservices
│   ├── api-gateway/   # Reverse proxy, route management
│   ├── auth-service/  # Authentication & authorization
│   ├── message-service/   # Real-time messaging with WebSocket
│   ├── location-service/  # Phone number geolocation
│   ├── user-service/      # User management (placeholder)
│   ├── media-service/     # Media handling (placeholder)
│   ├── notification-service/  # Push notifications (placeholder)
│   ├── presence-service/  # Online status (placeholder)
│   └── analytics-service/ # Analytics (placeholder)
├── shared/            # Shared libraries
│   ├── pkg/          # Infrastructure packages
│   └── server/       # HTTP server utilities
├── database/         # PostgreSQL schemas & migrations
│   └── schemas/      # Domain-specific SQL schemas
└── infra/            # Docker Compose, scripts
```

### Standard Service Structure

Every service follows this consistent pattern:

```
services/<service-name>/
├── cmd/server/main.go     # Entry point
├── internal/
│   ├── config/           # config.go, loader.go, validator.go
│   ├── handler/          # HTTP handlers
│   ├── service/          # Business logic (uses Builder pattern)
│   ├── repo/             # Data access layer
│   ├── health/           # Health check management
│   └── model/            # Domain models
├── configs/              # config.yaml, config.dev.yaml, config.prod.yaml
└── api/                  # API-specific code (if needed)
```

### Service Initialization Pattern

All services follow this initialization flow in `cmd/server/main.go`:

1. Load environment variables: `env.LoadEnv()`
2. Load configuration: `config.LoadWithEnv()` or `config.LoadFromEnv()`
3. Initialize structured logger (Zap)
4. Connect to infrastructure (DB, Cache, Kafka) with `defer` cleanup
5. Build service layer using Builder pattern
6. Setup health checks with custom checkers
7. Configure router with middleware chains
8. Create HTTP server with timeouts
9. Register graceful shutdown hooks with priorities
10. Start server and wait for OS signals

### Builder Pattern Usage

Services use the Builder pattern for construction and validation:

```go
// Service construction
authService := service.NewAuthServiceBuilder().
    WithRepo(authRepo).
    WithTokenService(tokenService).
    WithHashingService(hashingService).
    WithCache(cacheClient).
    WithConfig(&cfg.Auth).
    WithLogger(log).
    Build()  // Panics if required dependencies missing

// Router construction
router := router.NewBuilder().
    WithHealthEndpoint("/health", healthHandler).
    WithEarlyMiddleware(RequestID, CorrelationID, RequestLogger).
    WithLateMiddleware(Recovery, RequestCompletedLogger).
    WithRoutes(func(r *Router) { /* routes */ }).
    WithRoutesGroup("/api", func(rg *RouteGroup) { /* grouped routes */ }).
    Build()
```

**Important:** The `Build()` method validates that all required dependencies are provided. Missing dependencies will panic at startup with a clear error message.

## Configuration Management

### Hierarchical Configuration Loading

The system supports flexible configuration loading with three strategies:

1. **File-based:** `config.Load(path)` - Load from YAML file
2. **Environment-based:** `config.LoadFromEnv()` - Load from environment variables only
3. **Hybrid:** `config.LoadWithEnv(path, env)` - Base config + environment overlay

### Environment Variable Interpolation

Config files support environment variable interpolation with defaults:

```yaml
database:
  host: ${DB_HOST:localhost}
  port: ${DB_PORT:5432}
  user: ${DB_USER:echo}
```

### Standard Config Structure

```go
type Config struct {
    Service     ServiceConfig
    Server      ServerConfig
    Database    DatabaseConfig
    Cache       CacheConfig
    Security    SecurityConfig
    Logging     LoggingConfig
    Observability ObservabilityConfig
    Shutdown    ShutdownConfig
}
```

### Environment-Specific Configs

- `config.yaml` - Base configuration
- `config.dev.yaml` - Development overrides
- `config.prod.yaml` - Production overrides

Environment is selected via `APP_ENV` variable.

## Shared Packages

### Infrastructure Abstractions (`shared/pkg/`)

**Database (`shared/pkg/database`):**
- Interface-based with PostgreSQL implementation
- Models implement `TableName()` and `PrimaryKey()` methods
- Transaction support via `WithTransaction()`
- Operations: FindByID, FindAll, FindOne, Create, Update, Delete, RawQuery, RawExec

**Cache (`shared/pkg/cache`):**
- Redis-based with interface abstraction
- Operations: Get/Set/Delete, GetMulti/SetMulti, Increment/Decrement, Expire, TTL

**Messaging (`shared/pkg/messaging`):**
- Kafka producer/consumer abstractions
- Topic management, consumer groups

**Logger (`shared/pkg/logger`):**
- Structured logging interface with Zap adapter
- Methods: Info, Warn, Error, Debug, Fatal
- Context methods: String(), Int(), Error(), Any(), Duration()
- Special: `Request()` for HTTP logging

### Server Utilities (`shared/server/`)

**Router (`shared/server/router`):**
- Gorilla mux wrapper with fluent builder API
- Route groups with prefix support
- Method-specific routing: GET, POST, PUT, DELETE, PATCH

**Middleware (`shared/server/middleware`):**
- 15+ built-in middleware components
- Chain pattern: `middleware.NewChain().Append(...)`
- Early middleware (before route matching): RequestID, CorrelationID, RateLimiting
- Late middleware (after route matching): Recovery, RequestCompletedLogger

**Response (`shared/server/response`):**
- Standardized JSON responses
- Helper methods: OK, Created, NoContent, BadRequest, Unauthorized, NotFound, InternalServerError
- Error response with code and message

**Shutdown (`shared/server/shutdown`):**
- Priority-based graceful shutdown
- Priority levels: High (100), Normal (50), Low (10)
- Hooks: ServerShutdownHook, DelayHook, custom hooks

## Database Architecture

### Multi-Schema Design

The database uses 7 domain-specific schemas:
- `auth` - Authentication, tokens, sessions
- `users` - User profiles, contacts, blocking
- `messages` - Messages, conversations, delivery tracking
- `media` - Media files, thumbnails
- `notifications` - Push tokens, preferences
- `analytics` - Usage metrics, analytics
- `location` - Phone number geolocation

### Key Patterns

- **UUIDs:** All primary keys use UUID v4
- **Soft Deletes:** `deleted_at` timestamp for logical deletion
- **Timestamps:** Automatic `created_at`/`updated_at` via triggers
- **Audit Trail:** Full audit logging in `audit` schema
- **Row-Level Security:** RLS policies for access control

### Migration Management

Migrations are SQL files in `/database/schemas/<domain>/`. Use the provided scripts:
- `/infra/scripts/init-db.sh` - Initialize schemas
- `/infra/scripts/run-migrations.sh` - Run migrations

## Service-Specific Notes

### API Gateway

- Acts as reverse proxy for all services
- Routes configured in `services/api-gateway/configs/routes.yaml`
- Path transformation and prefix mapping supported
- Aggregates health checks from downstream services

### Auth Service

- Internal service (not exposed publicly)
- All auth requests go through API Gateway
- Uses phone-first authentication model
- JWT token generation and validation
- Redis for token blacklisting and rate limiting

### Message Service

- Real-time messaging via WebSocket (`/ws`)
- REST API for message operations
- Multi-device support (Hub manages user → [devices] mapping)
- Kafka for offline notifications
- Delivery tracking: sent → delivered → read
- Features: typing indicators, read receipts, message editing/deletion, mentions, threads

**WebSocket Connection:**
```
ws://localhost:8083/ws
Headers:
  X-User-ID: <uuid>
  X-Device-ID: <device-id>
  X-Platform: ios|android|web
```

**Message Flow:**
1. Client sends message via REST API
2. Service stores in DB and creates delivery records
3. Online users receive via WebSocket (auto-mark delivered)
4. Offline users receive via Kafka → notification-service

### Location Service

- Phone number geolocation lookup
- Returns country, region, timezone from phone number
- Used by auth-service for registration flow

## Development Workflow

### Adding a New Service

1. Create service directory: `services/<service-name>/`
2. Follow standard structure (cmd, internal, configs)
3. Implement Builder pattern for service initialization
4. Add Dockerfile and Dockerfile.dev
5. Update `docker-compose.dev.yml` with service definition
6. Add Makefile targets for service management
7. Create database schema in `database/schemas/<domain>/`
8. Register routes in API Gateway config (if needed)

### Adding a New Endpoint

1. Define route in handler setup (typically `internal/handler/<handler>.go`)
2. Implement handler method using service layer
3. Use standardized responses from `shared/server/response`
4. Add route to router builder in `cmd/server/main.go`
5. Apply appropriate middleware (auth, rate limiting, etc.)

### Configuration Changes

1. Update config struct in `internal/config/config.go`
2. Add validation in `internal/config/validator.go`
3. Update `configs/config.yaml` with new field
4. Add environment variable to `.env` and docker-compose files
5. Document in service-specific README (if applicable)

### Database Changes

1. Create migration SQL in `database/schemas/<domain>/`
2. Follow naming: `YYYYMMDD_description.up.sql` and `.down.sql`
3. Test locally: `make db-migrate`
4. Rollback test: `make db-migrate-down`
5. Update model structs in `internal/model/`

## Testing

### Running Tests

```bash
make test                    # All services
cd services/<service> && go test -v ./...  # Single service
cd shared && go test -v ./...              # Shared packages
```

### Test Organization

- Unit tests: `*_test.go` alongside source files
- Test coverage is currently limited (opportunity for expansion)
- Focus on business logic in service layer
- Infrastructure layer uses interfaces for mocking

## Common Pitfalls

1. **Missing Config Validation:** Always validate required config fields in `validator.go`
2. **Forgetting Defer Cleanup:** DB/Cache/Kafka clients must have `defer Close()`
3. **Shutdown Order:** Register shutdown hooks with correct priority (High → Normal → Low)
4. **Builder Pattern:** Don't forget to call `Build()` - it validates dependencies
5. **Database Transactions:** Always use `WithTransaction()` for multi-step operations
6. **Error Handling:** Use service-specific error codes, not generic errors
7. **Middleware Order:** Early middleware runs before routing, late middleware after

## Service URLs (Development)

- API Gateway: `http://localhost:8080`
- Auth Service: Internal only (via gateway at `/api/v1/auth/*`)
- Message Service: Internal only (via gateway at `/api/v1/messages/*`)
- Location Service: `http://localhost:8090` (direct access allowed)
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Kafka: `localhost:9092`

## Docker Compose Environments

- **Development:** `infra/docker/docker-compose.dev.yml` (hot reload enabled)
- **Production:** `infra/docker/docker-compose.prod.yml`

Set environment: `ENV=dev make up` or `ENV=prod make up`

## Go Workspace

The project uses Go workspaces (`go.work`) to manage multiple modules. All services and shared packages are in the workspace. Run `go work sync` after adding new dependencies.

## Key Dependencies

- **Web Framework:** Gorilla Mux (via shared/server/router)
- **Database:** PostgreSQL with pgx driver
- **Cache:** Redis
- **Messaging:** Kafka (Confluent)
- **Logging:** Zap
- **Configuration:** Viper
- **WebSocket:** Gorilla WebSocket (message-service)
- **Validation:** go-playground/validator

## Observability

### Health Checks

All services expose:
- `/health` - Liveness probe
- `/ready` - Readiness probe (checks dependencies)

Health checks verify:
- Database connectivity
- Cache connectivity (if enabled)
- Kafka connectivity (if applicable)

### Logging

- Structured JSON logging in production (Zap)
- Console logging in development
- Request/response logging via middleware
- Context-aware logging with correlation IDs

### Graceful Shutdown

Services handle SIGTERM/SIGINT:
1. Stop accepting new requests (High priority)
2. Drain existing connections (High priority)
3. Close infrastructure connections (Normal priority)
4. Sync logs and cleanup (Low priority)

Timeout: 30 seconds (configurable)