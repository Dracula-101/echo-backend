# Echo Backend

Echo Backend is a Go-based microservices backend for a real-time messaging platform. It provides an API gateway and a set of services for authentication, messaging, media handling, presence, and location lookup. The project is aimed at backend engineers who want a reference implementation or a starting point for building messaging systems.

## Documentation

- [Architecture](./docs/ARCHITECTURE.md)
- [API Reference](./docs/API_REFERENCE.md)
- [WebSocket Protocol](./docs/WEBSOCKET_PROTOCOL.md)
- [Usage Guide](./docs/USAGE.md)
- [Database Schema](./docs/DATABASE_SCHEMA.md)
- [Contributing](./CONTRIBUTING.md)

## Architecture & Entry Points

- **Microservices layout**: Each service lives under `services/<service>` with its entry point at `services/<service>/cmd/server/main.go` (see `go.work` for the full workspace list).
- **API Gateway**: Routes `/api/v1/*` traffic to internal services and enforces JWT validation and rate limiting (see `services/api-gateway/configs/config.yaml`).
- **WebSocket service**: Dedicated WebSocket server (`services/ws-service`) that upgrades connections at `/` and handles real-time events.
- **Shared libraries**: Common infrastructure lives in `shared/pkg` (database, cache, messaging, logging) and `shared/server` (router, middleware, health, shutdown, response envelope).

### Services

| Service | Purpose | Status |
| --- | --- | --- |
| api-gateway | Routing, JWT validation, rate limiting | Implemented |
| auth-service | Register, login, refresh tokens | Implemented |
| user-service | Profile creation, lookup, search | Implemented |
| message-service | Messages + conversations APIs | Implemented |
| presence-service | Presence and typing APIs | Implemented |
| media-service | Uploads, albums, file management | Implemented |
| location-service | IP-based location lookup | Implemented |
| ws-service | WebSocket real-time events | Implemented |
| notification-service | Push notifications | Placeholder (stub entry point) |
| analytics-service | Analytics | Placeholder (stub entry point) |

## Tech Stack

- **Language**: Go 1.25 (see `go.work` and service `go.mod` files)
- **HTTP**: Gorilla Mux
- **WebSocket**: Gorilla WebSocket
- **Database**: PostgreSQL (via `database/sql` + `lib/pq`)
- **Cache**: Redis (`go-redis`)
- **Messaging**: Kafka (IBM/Sarama)
- **Configuration**: Viper (YAML + env interpolation)
- **Logging**: Zap
- **Metrics**: Prometheus
- **Infra**: Docker + Docker Compose, Make

## Features

- API Gateway routing with JWT validation and rate limiting.
- Auth flows for registration, login, and refresh tokens.
- Messaging APIs for conversations and messages with Kafka integration.
- WebSocket service for real-time events (typing, presence, receipts, calls).
- Media upload and file management endpoints.
- Presence and typing indicator HTTP endpoints.
- Health, readiness, and metrics endpoints for observability.

## Project Structure

```
echo-backend/
├── services/              # Microservices (entry points under cmd/server)
├── shared/                # Shared libraries (db/cache/messaging/router/etc.)
├── database/              # SQL schemas, indexes, triggers, RLS
├── migrations/            # Database migration files
├── infra/                 # Docker Compose, scripts, and makefiles
├── scripts/               # Helper scripts (e.g., start-auth.sh)
├── docs/                  # Architecture and API docs
├── .env.example           # Root environment template
├── go.work                # Go workspace configuration
└── Makefile               # Development commands
```

## Getting Started

### Prerequisites

- Go 1.25
- Docker + Docker Compose
- Make

### Installation

```bash
git clone https://github.com/Dracula-101/echo-backend.git
cd echo-backend

# Root environment
cp .env.example .env

# Service environments
for service in api-gateway auth-service message-service user-service location-service media-service presence-service ws-service; do
  cp services/$service/.env.example services/$service/.env
done

# Start the stack
make up
```

### Database setup

Database initialization and migrations are handled by scripts under `infra/scripts` and make targets:

```bash
make db-init
make db-migrate
```

### Run locally

```bash
make up      # Start all services
make logs    # Follow logs
make health  # Health checks
make down    # Stop services
```

## Usage

### REST API (via API Gateway)

**Register**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "accept_terms": true
  }'
```

**Login**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

**Send a message**
```bash
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "660e8400-e29b-41d4-a716-446655440001",
    "content": "Hello!",
    "message_type": "text"
  }'
```

**Location lookup**
```bash
curl "http://localhost:8080/api/v1/location/lookup?ip=8.8.8.8"
```

Responses are wrapped in a standard envelope with `success`, `data`, and `error` fields (see `shared/server/response/response.go`).

### WebSocket

The WebSocket service listens at `/` (see `services/ws-service/cmd/server/main.go`). Provide the user ID via header or query param:

```
ws://localhost:8086/
Headers:
  X-User-ID: <uuid>
  X-Device-ID: <string>
  X-Platform: <string>
```

See [docs/WEBSOCKET_PROTOCOL.md](./docs/WEBSOCKET_PROTOCOL.md) for message formats and event types.

## Configuration

- **Root environment**: `.env.example` → `.env` (database, Redis, Kafka, JWT settings).
- **Service environments**: `services/*/.env.example` → `services/*/.env` (ports, logging, service-specific config).
- **Config files**: `services/<service>/configs/config.yaml` loaded via `CONFIG_PATH` and `APP_ENV`.

Key environment variables used across services:

- `POSTGRES_*`, `REDIS_*`, `KAFKA_BROKERS`
- `JWT_SECRET_KEY`, `JWT_ACCESS_TOKEN_TTL`, `JWT_REFRESH_TOKEN_TTL`
- `CONFIG_PATH`, `APP_ENV`, `LOG_LEVEL`, `LOG_FORMAT`

## Make Commands

```bash
make help        # List all commands
make up          # Start all services
make down        # Stop all services
make logs        # Tail logs
make health      # Health checks
make test        # Run tests
make db-init     # Initialize database
make db-migrate  # Run migrations
```

## Testing

```bash
make test
```

## Deployment

Docker Compose supports dev/prod overlays (see `infra/docker`):

```bash
ENV=prod make up
```

## Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for branching, style, and PR guidelines.

## License

No `LICENSE` file is present in the repository. `CONTRIBUTING.md` states that contributions are licensed under the MIT License.
