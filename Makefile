.PHONY: help up down build rebuild logs restart status clean ps health
.PHONY: gateway-up gateway-down gateway-logs gateway-restart gateway-build
.PHONY: auth-up auth-down auth-logs auth-restart auth-build
.PHONY: db-init db-seed db-connect db-reset redis-connect redis-flush
.PHONY: test setup

# Docker Compose files
COMPOSE_FILE = infra/docker/docker-compose.dev.yml

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

## help: Show this help message
help:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)  Echo Backend - Available Commands$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(GREEN)🚀 Main Services (API Gateway + Auth):$(NC)"
	@echo "  make up               - Start all services (gateway + auth + deps)"
	@echo "  make down             - Stop all services"
	@echo "  make restart          - Restart all services"
	@echo "  make build            - Build all service images"
	@echo "  make rebuild          - Rebuild all services (no cache)"
	@echo "  make logs             - View logs from all services"
	@echo "  make status           - Show status of all services"
	@echo "  make ps               - Show running containers"
	@echo "  make clean            - Stop and remove everything (including volumes)"	@echo "$(GREEN)🌐 API Gateway:$(NC)"
	@echo "  make gateway-up       - Start API Gateway"
	@echo "  make gateway-down     - Stop API Gateway"
	@echo "  make gateway-restart  - Restart API Gateway"
	@echo "  make gateway-build    - Rebuild API Gateway"
	@echo "  make gateway-logs     - View API Gateway logs"
	@echo ""
	@echo "$(GREEN)🔐 Auth Service:$(NC)"
	@echo "  make auth-up          - Start Auth Service"
	@echo "  make auth-down        - Stop Auth Service"
	@echo "  make auth-restart     - Restart Auth Service"
	@echo "  make auth-build       - Rebuild Auth Service"
	@echo "  make auth-logs        - View Auth Service logs"
	@echo ""
	@echo "$(GREEN)💾 Database:$(NC)"
	@echo "  make db-init          - Initialize database schemas"
	@echo "  make db-seed          - Seed database with test data"
	@echo "  make db-connect       - Connect to PostgreSQL"
	@echo "  make db-reset         - Reset database (drop and recreate)"
	@echo ""
	@echo "$(GREEN)📦 Redis:$(NC)"
	@echo "  make redis-connect    - Connect to Redis CLI"
	@echo "  make redis-flush      - Flush all Redis data"
	@echo ""
	@echo "$(GREEN)🛠️  Development:$(NC)"
	@echo "  make setup            - Initial setup (create .env, start services)"
	@echo "  make health           - Check health of all services"
	@echo "  make test             - Run tests"
	@echo ""

# =============================================================================
# Main Services (API Gateway + Auth Service)
# =============================================================================

## up: Start all services
up:
	@echo "$(GREEN)🚀 Starting all services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) up -d
	@echo ""
	@echo "$(GREEN)✓ Services started$(NC)"
	@echo "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BLUE)  API Gateway:$(NC)    http://localhost:8080"
	@echo "$(BLUE)  Auth Service:$(NC)   Internal only (via gateway)"
	@echo "$(BLUE)  PostgreSQL:$(NC)     localhost:5432"
	@echo "$(BLUE)  Redis:$(NC)          localhost:6379"
	@echo "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "💡 Useful commands:"
	@echo "   $(BLUE)make logs$(NC)    - View logs"
	@echo "   $(BLUE)make status$(NC)  - Check service status"
	@echo "   $(BLUE)make health$(NC)  - Run health checks"

## down: Stop all services
down:
	@echo "$(YELLOW)⏹️  Stopping all services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) down
	@echo "$(GREEN)✓ Services stopped$(NC)"

## restart: Restart all services
restart:
	@echo "$(YELLOW)🔄 Restarting all services...$(NC)"
	docker-compose -f $(COMPOSE_FILE) restart
	@echo "$(GREEN)✓ Services restarted$(NC)"

## build: Build all service images
build:
	@echo "$(GREEN)🔨 Building service images...$(NC)"
	docker-compose -f $(COMPOSE_FILE) build
	@echo "$(GREEN)✓ Build complete$(NC)"

## rebuild: Rebuild all services (no cache)
rebuild:
	@echo "$(GREEN)🔨 Rebuilding all services (no cache)...$(NC)"
	docker-compose -f $(COMPOSE_FILE) build --no-cache
	@echo "$(GREEN)✓ Rebuild complete$(NC)"

## logs: View logs from all services
logs:
	docker-compose -f $(COMPOSE_FILE) logs -f

## ps: Show running containers
ps:
	@docker-compose -f $(COMPOSE_FILE) ps

## status: Show status of all services
status:
	@echo "$(GREEN)📊 Service Status:$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) ps
	@echo ""

## clean: Stop and remove everything (including volumes)
clean:
	@echo "$(RED)🗑️  Cleaning up everything (this will delete all data)...$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f $(COMPOSE_FILE) down -v; \
		echo "$(GREEN)✓ Cleanup complete$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled$(NC)"; \
	fi

# =============================================================================
# API Gateway
# =============================================================================

## gateway-up: Start API Gateway
gateway-up:
	@echo "$(GREEN)🌐 Starting API Gateway...$(NC)"
	docker-compose -f $(COMPOSE_FILE) up -d api-gateway
	@echo "$(GREEN)✓ API Gateway started$(NC)"
	@echo "$(YELLOW)Gateway: http://localhost:8080$(NC)"

## gateway-down: Stop API Gateway
gateway-down:
	@echo "$(YELLOW)Stopping API Gateway...$(NC)"
	docker-compose -f $(COMPOSE_FILE) stop api-gateway
	@echo "$(GREEN)✓ API Gateway stopped$(NC)"

## gateway-restart: Restart API Gateway
gateway-restart:
	@echo "$(YELLOW)🔄 Restarting API Gateway...$(NC)"
	docker-compose -f $(COMPOSE_FILE) restart api-gateway
	@echo "$(GREEN)✓ API Gateway restarted$(NC)"

## gateway-build: Rebuild API Gateway
gateway-build:
	@echo "$(GREEN)🔨 Rebuilding API Gateway...$(NC)"
	docker-compose -f $(COMPOSE_FILE) build --no-cache api-gateway
	@echo "$(GREEN)✓ Build complete$(NC)"

## gateway-logs: View API Gateway logs
gateway-logs:
	docker-compose -f $(COMPOSE_FILE) logs -f api-gateway

# =============================================================================
# Auth Service
# =============================================================================

## auth-up: Start Auth Service
auth-up:
	@echo "$(GREEN)🔐 Starting Auth Service...$(NC)"
	docker-compose -f $(COMPOSE_FILE) up -d auth-service
	@echo "$(GREEN)✓ Auth Service started$(NC)"

## auth-down: Stop Auth Service
auth-down:
	@echo "$(YELLOW)Stopping Auth Service...$(NC)"
	docker-compose -f $(COMPOSE_FILE) stop auth-service
	@echo "$(GREEN)✓ Auth Service stopped$(NC)"

## auth-restart: Restart Auth Service
auth-restart:
	@echo "$(YELLOW)🔄 Restarting Auth Service...$(NC)"
	docker-compose -f $(COMPOSE_FILE) restart auth-service
	@echo "$(GREEN)✓ Auth Service restarted$(NC)"

## auth-build: Rebuild Auth Service
auth-build:
	@echo "$(GREEN)🔨 Rebuilding Auth Service...$(NC)"
	docker-compose -f $(COMPOSE_FILE) build --no-cache auth-service
	@echo "$(GREEN)✓ Build complete$(NC)"

## auth-logs: View Auth Service logs
auth-logs:
	docker-compose -f $(COMPOSE_FILE) logs -f auth-service

# =============================================================================
# Database
# =============================================================================
db-init:
	@echo "$(GREEN)Initializing database...$(NC)"
	@cd infra/scripts && chmod +x init-db.sh && ./init-db.sh
	@echo "$(GREEN)✓ Database initialized$(NC)"

## db-seed: Seed database with test data
db-seed:
	@echo "$(GREEN)Seeding database...$(NC)"
	@cd infra/scripts && chmod +x seed-data.sh && ./seed-data.sh
	@echo "$(GREEN)✓ Database seeded$(NC)"
	@echo "$(YELLOW)Test accounts:$(NC)"
	@echo "  - alice@example.com"
	@echo "  - bob@example.com"
	@echo "  - charlie@example.com"

## db-connect: Connect to PostgreSQL
db-connect:
	@echo "$(GREEN)Connecting to PostgreSQL...$(NC)"
	docker exec -it echo-postgres psql -U echo -d echo_db

## db-reset: Reset database (drop and recreate)
db-reset:
	@echo "$(RED)Resetting database (this will delete all data)...$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -it echo-postgres psql -U echo -c "DROP DATABASE IF EXISTS echo_db;"; \
		docker exec -it echo-postgres psql -U echo -c "CREATE DATABASE echo_db;"; \
		$(MAKE) db-init; \
		echo "$(GREEN)✓ Database reset complete$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled$(NC)"; \
	fi

## redis-connect: Connect to Redis CLI
redis-connect:
	@echo "$(GREEN)Connecting to Redis...$(NC)"
	docker exec -it echo-redis redis-cli -a redis_password

## redis-flush: Flush all Redis data
redis-flush:
	@echo "$(RED)🗑️  Flushing Redis data...$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -it echo-redis redis-cli -a redis_password FLUSHALL; \
		echo "$(GREEN)✓ Redis flushed$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled$(NC)"; \
	fi

# =============================================================================
# Development & Testing
# =============================================================================

## setup: Initial setup
setup:
	@echo "$(GREEN)🛠️  Setting up Echo Backend...$(NC)"
	@if [ ! -f services/api-gateway/.env ]; then \
		touch services/api-gateway/.env; \
		echo "$(GREEN)✓ Created api-gateway .env file$(NC)"; \
	else \
		echo "$(YELLOW)⚠ api-gateway .env file already exists$(NC)"; \
	fi
	@if [ ! -f services/auth-service/.env ]; then \
		touch services/auth-service/.env; \
		echo "$(GREEN)✓ Created auth-service .env file$(NC)"; \
	else \
		echo "$(YELLOW)⚠ auth-service .env file already exists$(NC)"; \
	fi
	@$(MAKE) up
	@sleep 5
	@echo ""
	@echo "$(GREEN)✓ Setup complete!$(NC)"
	@echo "$(YELLOW)Run 'make health' to verify all services$(NC)"

## health: Check health of all services
health:
	@echo "$(GREEN)🏥 Checking service health...$(NC)"
	@echo ""
	@echo "$(BLUE)API Gateway:$(NC)"
	@curl -f http://localhost:8080/health >/dev/null 2>&1 && \
		echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo ""
	@echo "$(BLUE)Auth Service:$(NC)"
	@docker exec echo-api-gateway curl -f http://auth-service:8081/health >/dev/null 2>&1 && \
		echo "$(GREEN)✓ Healthy$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo ""
	@echo "$(BLUE)PostgreSQL:$(NC)"
	@docker exec echo-postgres pg_isready -U echo 2>/dev/null && echo "$(GREEN)✓ Ready$(NC)" || echo "$(RED)✗ Not ready$(NC)"
	@echo ""
	@echo "$(BLUE)Redis:$(NC)"
	@docker exec echo-redis redis-cli -a redis_password PING 2>/dev/null | grep -q PONG && echo "$(GREEN)✓ Ready$(NC)" || echo "$(RED)✗ Not responding$(NC)"
	@echo ""

## test: Run tests
test:
	@echo "$(GREEN)🧪 Running tests...$(NC)"
	@cd services/auth-service && go test -v ./...
	@cd services/api-gateway && go test -v ./...

# =============================================================================
# Utility Commands
# =============================================================================

## test-auth: Test auth endpoints through gateway
test-auth:
	@echo "$(GREEN)Testing auth endpoints...$(NC)"
	@echo ""
	@echo "$(BLUE)Testing /login endpoint:$(NC)"
	@curl -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"test123"}' \
		2>/dev/null | jq .
	@echo ""

## verify-security: Verify auth service is not exposed
verify-security:
	@echo "$(GREEN)🔒 Verifying security configuration...$(NC)"
	@echo ""
	@echo "$(BLUE)Testing direct access to auth-service (should fail):$(NC)"
	@(curl -X POST http://localhost:8081/login -m 2 2>&1 | grep -q "Connection refused" || \
	 curl -X POST http://localhost:8081/login -m 2 2>&1 | grep -q "Failed to connect") && \
		echo "$(GREEN)✓ Auth service is properly secured (not accessible directly)$(NC)" || \
		echo "$(RED)✗ WARNING: Auth service is exposed on port 8081$(NC)"
	@echo ""
	@echo "$(BLUE)Testing access via API Gateway (should work):$(NC)"
	@curl -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"test123"}' \
		-s -o /dev/null -w "%{http_code}" | grep -q "200" && \
		echo "$(GREEN)✓ Gateway proxy is working correctly$(NC)" || \
		echo "$(RED)✗ Gateway proxy is not working$(NC)"

## dev: Start services in development mode
dev: up
	@echo "$(GREEN)🚀 Development mode started$(NC)"
	@$(MAKE) logs
## dev: Start services in development mode
dev: up
	@echo "$(GREEN)🚀 Development mode started$(NC)"
	@$(MAKE) logs

# =============================================================================
# Legacy Commands (for backward compatibility)
# =============================================================================

generate-routes:
	@echo "$(GREEN)Generating routes from registry...$(NC)"
	@go run scripts/generate-routes.go shared/routes/registry.yaml
	@echo "$(GREEN)✓ Routes generated successfully$(NC)"

update-gateway-routes: generate-routes
	@echo "$(GREEN)Updating gateway configuration...$(NC)"
	@cp shared/routes/registry.yaml services/api-gateway/configs/routes.yaml
	@echo "$(GREEN)✓ Gateway routes updated$(NC)"
