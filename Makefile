# =============================================================================
# ECHO BACKEND - MAKEFILE
# =============================================================================
# Organized Makefile for managing Echo Backend microservices
# =============================================================================

.PHONY: help up down build rebuild logs restart status clean ps health
.PHONY: gateway-up gateway-down gateway-logs gateway-restart gateway-build gateway-rerun gateway-rebuild
.PHONY: auth-up auth-down auth-logs auth-restart auth-build auth-rerun auth-rebuild
.PHONY: user-up user-down user-logs user-restart user-build user-rerun user-rebuild
.PHONY: message-up message-down message-logs message-restart message-build message-rerun message-rebuild
.PHONY: location-up location-down location-logs location-restart location-build location-rerun location-rebuild
.PHONY: media-up media-down media-logs media-restart media-build media-rerun media-rebuild
.PHONY: kafka-up kafka-down kafka-logs kafka-restart kafka-topics kafka-create-topics
.PHONY: db-up db-init db-seed db-connect db-clean db-reset db-migrate db-migrate-down db-migrate-status
.PHONY: redis-connect redis-flush test setup test-auth verify-security dev generate-routes update-gateway-routes

# =============================================================================
# CONFIGURATION
# =============================================================================

# Environment Selection (dev/prod)
ENV ?= dev

# Docker Compose files
COMPOSE_FILE_DEV := infra/docker/docker-compose.dev.yml
COMPOSE_FILE_PROD := infra/docker/docker-compose.prod.yml

# Select compose file based on environment
ifeq ($(ENV),prod)
    COMPOSE_FILE := $(COMPOSE_FILE_PROD)
    ENV_NAME := Production
else
    COMPOSE_FILE := $(COMPOSE_FILE_DEV)
    ENV_NAME := Development
endif

# Default target
.DEFAULT_GOAL := help

# Colors
BOLD := \033[1m
DIM := \033[2m
BLUE := \033[38;5;33m
GREEN := \033[38;5;82m
YELLOW := \033[38;5;220m
RED := \033[38;5;196m
PURPLE := \033[38;5;141m
CYAN := \033[38;5;51m
GRAY := \033[38;5;240m
NC := \033[0m

# Symbols
CHECK := ✓
CROSS := ✗

# =============================================================================
# HELP
# =============================================================================

## help: Show this help message
help:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)$(CYAN)  ECHO BACKEND $(NC)$(GRAY)·$(NC) $(DIM)Microservices Management$(NC)"
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)Main Services:$(NC)"
	@echo "  make up               Start all services"
	@echo "  make down             Stop all services"
	@echo "  make restart          Restart all services"
	@echo "  make build            Build all service images"
	@echo "  make rerun            Stop, rebuild and start all services"
	@echo "  make logs             View logs from all services"
	@echo "  make status           Show status of all services"
	@echo "  make ps               Show running containers"
	@echo "  make clean            Stop and remove everything (including volumes)"
	@echo ""
	@echo "$(BOLD)API Gateway:$(NC)"
	@echo "  make gateway-up       Start API Gateway"
	@echo "  make gateway-down     Stop API Gateway"
	@echo "  make gateway-rerun    Stop and restart API Gateway"
	@echo "  make gateway-restart  Restart API Gateway"
	@echo "  make gateway-build    Rebuild API Gateway"
	@echo "  make gateway-rebuild  Rebuild API Gateway (no cache)"
	@echo "  make gateway-logs     View API Gateway logs"
	@echo ""
	@echo "$(BOLD)Auth Service:$(NC)"
	@echo "  make auth-up          Start Auth Service"
	@echo "  make auth-down        Stop Auth Service"
	@echo "  make auth-rerun       Stop and restart Auth Service"
	@echo "  make auth-restart     Restart Auth Service"
	@echo "  make auth-build       Rebuild Auth Service"
	@echo "  make auth-rebuild     Rebuild Auth Service (no cache)"
	@echo "  make auth-logs        View Auth Service logs"
	@echo ""
	@echo "$(BOLD)Location Service:$(NC)"
	@echo "  make location-up      Start Location Service"
	@echo "  make location-down    Stop Location Service"
	@echo "  make location-rerun   Stop and restart Location Service"
	@echo "  make location-restart Restart Location Service"
	@echo "  make location-build   Rebuild Location Service"
	@echo "  make location-rebuild Rebuild Location Service (no cache)"
	@echo "  make location-logs    View Location Service logs"
	@echo ""
	@echo "$(BOLD)User Service:$(NC)"
	@echo "  make user-up          Start User Service"
	@echo "  make user-down        Stop User Service"
	@echo "  make user-rerun       Stop and restart User Service"
	@echo "  make user-restart     Restart User Service"
	@echo "  make user-build       Rebuild User Service"
	@echo "  make user-rebuild     Rebuild User Service (no cache)"
	@echo "  make user-logs        View User Service logs"
	@echo ""
	@echo "$(BOLD)Message Service:$(NC)"
	@echo "  make message-up       Start Message Service"
	@echo "  make message-down     Stop Message Service"
	@echo "  make message-rerun    Stop and restart Message Service"
	@echo "  make message-restart  Restart Message Service"
	@echo "  make message-build    Rebuild Message Service"
	@echo "  make message-rebuild  Rebuild Message Service (no cache)"
	@echo "  make message-logs     View Message Service logs"
	@echo ""
	@echo "$(BOLD)Media Service:$(NC)"
	@echo "  make media-up         Start Media Service"
	@echo "  make media-down       Stop Media Service"
	@echo "  make media-rerun      Stop and restart Media Service"
	@echo "  make media-restart    Restart Media Service"
	@echo "  make media-build      Rebuild Media Service"
	@echo "  make media-rebuild    Rebuild Media Service (no cache)"
	@echo "  make media-logs       View Media Service logs"
	@echo ""
	@echo "$(BOLD)Kafka:$(NC)"
	@echo "  make kafka-up         Start Kafka and Zookeeper"
	@echo "  make kafka-down       Stop Kafka and Zookeeper"
	@echo "  make kafka-logs       View Kafka logs"
	@echo "  make kafka-restart    Restart Kafka"
	@echo "  make kafka-topics     List Kafka topics"
	@echo "  make kafka-create-topics  Create required Kafka topics"
	@echo ""
	@echo "$(BOLD)Database:$(NC)"
	@echo "  make db-up            Start PostgreSQL"
	@echo "  make db-init          Initialize database schemas"
	@echo "  make db-seed          Seed database with test data"
	@echo "  make db-connect       Connect to PostgreSQL"
	@echo "  make db-reset         Reset database (drop and recreate)"
	@echo "  make db-migrate       Run database migrations"
	@echo "  make db-migrate-down  Rollback last migration"
	@echo "  make db-migrate-status Show migration status"
	@echo ""
	@echo "$(BOLD)Redis:$(NC)"
	@echo "  make redis-connect    Connect to Redis CLI"
	@echo "  make redis-flush      Flush all Redis data"
	@echo ""
	@echo "$(BOLD)Development:$(NC)"
	@echo "  make setup            Initial setup (create .env, start services)"
	@echo "  make dev              Start in development mode with logs"
	@echo "  make health           Check health of all services"
	@echo "  make test             Run tests"
	@echo "  make test-auth        Test auth endpoints"
	@echo "  make verify-security  Verify auth service security"
	@echo ""
	@echo "$(BOLD)Utilities:$(NC)"
	@echo "  make generate-routes  Generate routes from registry"
	@echo "  make update-gateway-routes Update gateway routes"
	@echo ""
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""

# =============================================================================
# MAIN SERVICES
# =============================================================================

## up: Start all services
up:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting All Services$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d
	@sleep 2
	@echo ""
	@echo "$(GREEN)$(CHECK) Services started successfully$(NC)"
	@echo ""
	@echo "$(BOLD)Service URLs:$(NC)"
	@echo "  API Gateway      	$(CYAN)http://localhost:8080$(NC)"
	@echo "  Auth Service     	$(GRAY)Internal only (via gateway)$(NC)"
	@echo "  User Service     	$(GRAY)Internal only (via gateway)$(NC)"
	@echo "  Messages Service 	$(GRAY)Internal only (via gateway)$(NC)"
	@echo "  Media Service    	$(GRAY)Internal only (via gateway)$(NC)"
	@echo "  Location Service 	$(CYAN)http://localhost:8090$(NC)"
	@echo "  WebSocket        	$(CYAN)ws://localhost:8083/ws$(NC)"
	@echo ""
	@echo "$(BOLD)Infrastructure:$(NC)"
	@echo "  PostgreSQL       $(YELLOW)localhost:5432$(NC)"
	@echo "  Redis            $(YELLOW)localhost:6379$(NC)"
	@echo "  Kafka            $(YELLOW)localhost:9092$(NC)"
	@echo ""
	@echo "$(BOLD)Quick Commands:$(NC)"
	@echo "  $(CYAN)make logs$(NC)    View live logs"
	@echo "  $(CYAN)make status$(NC)  Check service status"
	@echo "  $(CYAN)make health$(NC)  Run health checks"
	@echo ""

## down: Stop all services
down:
	@echo ""
	@echo "$(BOLD)$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Stopping Services$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) down
	@echo ""
	@echo "$(GREEN)$(CHECK) All services stopped$(NC)"
	@echo ""

## restart: Restart all services
restart:
	@echo ""
	@echo "$(BOLD)$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Restarting Services$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) restart
	@echo ""
	@echo "$(GREEN)$(CHECK) Services restarted$(NC)"
	@echo ""

## build: Build all service images
build:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building Service Images$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## rerun: Stop, rebuild and start all services
rerun:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuild and Restart$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(DIM)Step 1/3: Stopping services...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) down
	@echo "$(DIM)Step 2/3: Building images...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) build --no-cache
	@echo "$(DIM)Step 3/3: Starting services...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) up -d
	@echo ""
	@echo "$(GREEN)$(CHECK) Services rebuilt and restarted$(NC)"
	@echo ""

## logs: View logs from all services
logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f --no-log-prefix

## ps: Show running containers
ps:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Running Containers$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) ps
	@echo ""

## status: Show status of all services
status:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Service Status$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)API Gateway$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-api-gateway 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-api-gateway 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)Auth Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-auth-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-auth-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)User Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-user-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-user-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)Message Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-message-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-message-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)Media Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-media-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-media-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)Location Service$(NC)"
	@docker inspect -f '{{.State.Running}}' location-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' location-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)Kafka$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-kafka 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-kafka 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)PostgreSQL$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-postgres 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-postgres 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)Redis$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-redis 2>/dev/null | grep -q true && \
		echo "  Status:  $(GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-redis 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""

## clean: Stop and remove everything (including volumes)
clean:
	@echo ""
	@echo "$(BOLD)$(RED)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Cleanup $(NC)$(RED)$(CROSS) This will delete all data!$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f $(COMPOSE_FILE) down -v; \
		docker-compose -f $(COMPOSE_FILE) rm -f; \
		docker system prune -f; \
		docker volume prune -f; \
		docker network prune -f; \
		docker image prune -f; \
		docker volume rm $$(docker volume ls -qf dangling=true) 2>/dev/null || true; \
		docker network rm $$(docker network ls -qf dangling=true) 2>/dev/null || true; \
		docker image rm $$(docker images -qf dangling=true) 2>/dev/null || true; \
		echo ""; \
		echo "$(GREEN)$(CHECK) Cleanup complete$(NC)"; \
		echo ""; \
	else \
		echo ""; \
		echo "$(YELLOW)Cancelled$(NC)"; \
		echo ""; \
	fi

# =============================================================================
# API GATEWAY
# =============================================================================

## gateway-up: Start API Gateway
gateway-up:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting API Gateway$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d api-gateway
	@echo ""
	@echo "$(GREEN)$(CHECK) API Gateway started$(NC)"
	@echo "  URL: $(CYAN)http://localhost:8080$(NC)"
	@echo ""

## gateway-down: Stop API Gateway
gateway-down:
	@echo ""
	@echo "$(YELLOW)Stopping API Gateway...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop api-gateway
	@echo "$(GREEN)$(CHECK) API Gateway stopped$(NC)"
	@echo ""

## gateway-rerun: Stop and restart API Gateway
gateway-rerun:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rerunning API Gateway$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) stop api-gateway
	@docker-compose -f $(COMPOSE_FILE) up -d api-gateway
	@echo ""
	@echo "$(GREEN)$(CHECK) API Gateway rerun complete$(NC)"
	@echo ""

## gateway-restart: Restart API Gateway
gateway-restart:
	@echo ""
	@echo "$(YELLOW)Restarting API Gateway...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart api-gateway
	@echo "$(GREEN)$(CHECK) API Gateway restarted$(NC)"
	@echo ""

## gateway-build: Rebuild API Gateway
gateway-build:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building API Gateway$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build api-gateway
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## gateway-rebuild: Rebuild API Gateway (no cache)
gateway-rebuild:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuilding API Gateway $(DIM)(no cache)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build --no-cache api-gateway
	@echo ""
	@echo "$(GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

## gateway-logs: View API Gateway logs
gateway-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  API Gateway Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f api-gateway --no-log-prefix

# =============================================================================
# AUTH SERVICE
# =============================================================================

## auth-up: Start Auth Service
auth-up:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting Auth Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d auth-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Auth Service started$(NC)"
	@echo ""

## auth-down: Stop Auth Service
auth-down:
	@echo ""
	@echo "$(YELLOW)Stopping Auth Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop auth-service
	@echo "$(GREEN)$(CHECK) Auth Service stopped$(NC)"
	@echo ""

## auth-rerun: Stop and restart Auth Service
auth-rerun:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rerunning Auth Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) stop auth-service
	@docker-compose -f $(COMPOSE_FILE) up -d auth-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Auth Service rerun complete$(NC)"
	@echo ""

## auth-restart: Restart Auth Service
auth-restart:
	@echo ""
	@echo "$(YELLOW)Restarting Auth Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart auth-service
	@echo "$(GREEN)$(CHECK) Auth Service restarted$(NC)"
	@echo ""

## auth-build: Rebuild Auth Service
auth-build:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building Auth Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build auth-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## auth-rebuild: Rebuild Auth Service (no cache)
auth-rebuild:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuilding Auth Service $(DIM)(no cache)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build --no-cache auth-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

## auth-logs: View Auth Service logs
auth-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Auth Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f auth-service --no-log-prefix

# =============================================================================
# USER SERVICE
# =============================================================================

## user-up: Start User Service
user-up:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting User Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d user-service
	@echo ""
	@echo "$(GREEN)$(CHECK) User Service started$(NC)"
	@echo ""

## user-down: Stop User Service
user-down:
	@echo ""
	@echo "$(YELLOW)Stopping User Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop user-service
	@echo "$(GREEN)$(CHECK) User Service stopped$(NC)"
	@echo ""

## user-rerun: Stop and restart User Service
user-rerun:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rerunning User Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) stop user-service
	@docker-compose -f $(COMPOSE_FILE) up -d user-service
	@echo ""
	@echo "$(GREEN)$(CHECK) User Service rerun complete$(NC)"
	@echo ""

## user-restart: Restart User Service
user-restart:
	@echo ""
	@echo "$(YELLOW)Restarting User Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart user-service
	@echo "$(GREEN)$(CHECK) User Service restarted$(NC)"
	@echo ""

## user-build: Rebuild User Service
user-build:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building User Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build user-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## user-rebuild: Rebuild User Service (no cache)
user-rebuild:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuilding User Service $(DIM)(no cache)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build --no-cache user-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

## user-logs: View User Service logs
user-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  User Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f user-service --no-log-prefix

# =============================================================================
# MESSAGE SERVICE
# =============================================================================

## message-up: Start Message Service
message-up:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting Message Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d message-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Message Service started$(NC)"
	@echo "  REST API:   $(CYAN)http://localhost:8083$(NC)"
	@echo "  WebSocket:  $(CYAN)ws://localhost:8083/ws$(NC)"
	@echo ""

## message-down: Stop Message Service
message-down:
	@echo ""
	@echo "$(YELLOW)Stopping Message Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop message-service
	@echo "$(GREEN)$(CHECK) Message Service stopped$(NC)"
	@echo ""

## message-rerun: Stop and restart Message Service
message-rerun:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rerunning Message Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) stop message-service
	@docker-compose -f $(COMPOSE_FILE) up -d message-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Message Service rerun complete$(NC)"
	@echo ""

## message-restart: Restart Message Service
message-restart:
	@echo ""
	@echo "$(YELLOW)Restarting Message Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart message-service
	@echo "$(GREEN)$(CHECK) Message Service restarted$(NC)"
	@echo ""

## message-build: Rebuild Message Service
message-build:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building Message Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build message-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## message-rebuild: Rebuild Message Service (no cache)
message-rebuild:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuilding Message Service $(DIM)(no cache)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build --no-cache message-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

## message-logs: View Message Service logs
message-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Message Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f message-service --no-log-prefix

# =============================================================================
# LOCATION SERVICE
# =============================================================================

## location-up: Start Location Service
location-up:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting Location Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d location-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Location Service started$(NC)"
	@echo ""

## location-down: Stop Location Service
location-down:
	@echo ""
	@echo "$(YELLOW)Stopping Location Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop location-service
	@echo "$(GREEN)$(CHECK) Location Service stopped$(NC)"
	@echo ""

## location-rerun: Stop and restart Location Service
location-rerun:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rerunning Location Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) stop location-service
	@docker-compose -f $(COMPOSE_FILE) up -d location-service --build --force-recreate
	@echo ""
	@echo "$(GREEN)$(CHECK) Location Service rerun complete$(NC)"
	@echo ""

## location-restart: Restart Location Service
location-restart:
	@echo ""
	@echo "$(YELLOW)Restarting Location Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart location-service
	@echo "$(GREEN)$(CHECK) Location Service restarted$(NC)"
	@echo ""

## location-build: Rebuild Location Service
location-build:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building Location Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build location-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## location-rebuild: Rebuild Location Service (no cache)
location-rebuild:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuilding Location Service $(DIM)(no cache)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build --no-cache location-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

## location-logs: View Location Service logs
location-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Location Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f location-service --no-log-prefix

# =============================================================================
# MEDIA SERVICE
# =============================================================================

## media-up: Start Media Service
media-up:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting Media Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d media-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Media Service started$(NC)"
	@echo ""

## media-down: Stop Media Service
media-down:
	@echo ""
	@echo "$(YELLOW)Stopping Media Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop media-service
	@echo "$(GREEN)$(CHECK) Media Service stopped$(NC)"
	@echo ""

## media-rerun: Stop and restart Media Service
media-rerun:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rerunning Media Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) stop media-service
	@docker-compose -f $(COMPOSE_FILE) up -d media-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Media Service rerun complete$(NC)"
	@echo ""

## media-restart: Restart Media Service
media-restart:
	@echo ""
	@echo "$(YELLOW)Restarting Media Service...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart media-service
	@echo "$(GREEN)$(CHECK) Media Service restarted$(NC)"
	@echo ""

## media-build: Rebuild Media Service
media-build:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Building Media Service$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build media-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

## media-rebuild: Rebuild Media Service (no cache)
media-rebuild:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rebuilding Media Service $(DIM)(no cache)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) build --no-cache media-service
	@echo ""
	@echo "$(GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

## media-logs: View Media Service logs
media-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Media Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f media-service --no-log-prefix

# =============================================================================
# KAFKA & ZOOKEEPER
# =============================================================================

## kafka-up: Start Kafka and Zookeeper
kafka-up:
	@echo ""
	@echo "$(BOLD)$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting Kafka$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d zookeeper kafka
	@echo ""
	@echo "$(GREEN)$(CHECK) Kafka started$(NC)"
	@echo "  URL: $(CYAN)localhost:9092$(NC)"
	@echo ""

## kafka-down: Stop Kafka and Zookeeper
kafka-down:
	@echo ""
	@echo "$(YELLOW)Stopping Kafka...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) stop kafka zookeeper
	@echo "$(GREEN)$(CHECK) Kafka stopped$(NC)"
	@echo ""

## kafka-logs: View Kafka logs
kafka-logs:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Kafka Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) logs -f kafka

## kafka-restart: Restart Kafka
kafka-restart:
	@echo ""
	@echo "$(YELLOW)Restarting Kafka...$(NC)"
	@docker-compose -f $(COMPOSE_FILE) restart kafka
	@echo "$(GREEN)$(CHECK) Kafka restarted$(NC)"
	@echo ""

## kafka-topics: List Kafka topics
kafka-topics:
	@echo ""
	@echo "$(BOLD)$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Kafka Topics$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker exec echo-kafka kafka-topics --list --bootstrap-server localhost:9092
	@echo ""

## kafka-create-topics: Create required Kafka topics
kafka-create-topics:
	@echo ""
	@echo "$(BOLD)$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Creating Kafka Topics$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker exec echo-kafka kafka-topics --create --if-not-exists --bootstrap-server localhost:9092 --topic messages --partitions 3 --replication-factor 1
	@docker exec echo-kafka kafka-topics --create --if-not-exists --bootstrap-server localhost:9092 --topic notifications --partitions 3 --replication-factor 1
	@echo ""
	@echo "$(GREEN)$(CHECK) Topics created$(NC)"
	@echo ""

# =============================================================================
# DATABASE
# =============================================================================

## db-up: Start PostgreSQL
db-up:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Starting PostgreSQL$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker-compose -f $(COMPOSE_FILE) up -d postgres
	@echo ""
	@echo "$(GREEN)$(CHECK) PostgreSQL started$(NC)"
	@echo "  URL: $(CYAN)localhost:5432$(NC)"
	@echo "  User: $(CYAN)echo$(NC)"
	@echo "  DB:   $(CYAN)echo_db$(NC)"
	@echo ""

## db-init: Initialize database
db-init:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Initializing Database$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x init-db.sh && ./init-db.sh --force
	@echo ""
	@echo "$(GREEN)$(CHECK) Database initialized$(NC)"
	@echo ""

## db-seed: Seed database with test data
db-seed:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Seeding Database$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x seed-data.sh && ./seed-data.sh
	@echo ""
	@echo "$(GREEN)$(CHECK) Database seeded$(NC)"
	@echo ""
	@echo "$(BOLD)Test Accounts:$(NC)"
	@echo "  $(GREEN)$(CHECK)$(NC) alice@example.com"
	@echo "  $(GREEN)$(CHECK)$(NC) bob@example.com"
	@echo "  $(GREEN)$(CHECK)$(NC) charlie@example.com"
	@echo ""

## db-connect: Connect to PostgreSQL
db-connect:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Connecting to PostgreSQL$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker exec -it echo-postgres psql -U echo -d echo_db

# db-clean: Remove all the data from the database
db-clean:
	@echo ""
	@echo "$(BOLD)$(RED)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Database Clean $(NC)$(RED)$(CROSS) This will delete all data!$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd infra/scripts && chmod +x clean-db.sh && ./clean-db.sh; \
		echo ""; \
		echo "$(GREEN)$(CHECK) Database cleaned$(NC)"; \
		echo ""; \
	else \
		echo ""; \
		echo "$(YELLOW)Cancelled$(NC)"; \
		echo ""; \
	fi

## db-reset: Reset database (drop and recreate)
db-reset:
	@echo ""
	@echo "$(BOLD)$(RED)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Database Reset $(NC)$(RED)$(CROSS) This will delete all data!$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		if docker exec -it echo-postgres psql -U echo -c "DROP DATABASE IF EXISTS echo_db;" 2>/dev/null; then \
			if docker exec -it echo-postgres psql -U echo -c "CREATE DATABASE echo_db;" 2>/dev/null; then \
				if $(MAKE) db-init; then \
					echo ""; \
					echo "$(GREEN)$(CHECK) Database reset complete$(NC)"; \
					echo ""; \
				else \
					echo ""; \
					echo "$(RED)$(CROSS) Failed to initialize database$(NC)"; \
					echo ""; \
					exit 1; \
				fi; \
			else \
				echo ""; \
				echo "$(RED)$(CROSS) Failed to create database$(NC)"; \
				echo ""; \
				exit 1; \
			fi; \
		else \
			echo ""; \
			echo "$(RED)$(CROSS) Failed to drop database$(NC)"; \
			echo ""; \
			exit 1; \
		fi; \
	else \
		echo ""; \
		echo "$(YELLOW)Cancelled$(NC)"; \
		echo ""; \
	fi

## db-migrate: Run database migrations
db-migrate:
	@echo ""
	@echo "$(BOLD)$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Running Migrations$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x run-migrations.sh && ./run-migrations.sh up
	@echo ""
	@echo "$(GREEN)$(CHECK) Migrations applied$(NC)"
	@echo ""

## db-migrate-down: Rollback last migration
db-migrate-down:
	@echo ""
	@echo "$(BOLD)$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Rolling Back Migration$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x run-migrations.sh && ./run-migrations.sh down
	@echo ""
	@echo "$(GREEN)$(CHECK) Migration rolled back$(NC)"
	@echo ""

## db-migrate-status: Show migration status
db-migrate-status:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Migration Status$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x run-migrations.sh && ./run-migrations.sh status
	@echo ""

# =============================================================================
# REDIS
# =============================================================================

## redis-connect: Connect to Redis CLI
redis-connect:
	@echo ""
	@echo "$(BOLD)$(RED)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Connecting to Redis$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@docker exec -it echo-redis redis-cli -a redis_password

## redis-flush: Flush all Redis data
redis-flush:
	@echo ""
	@echo "$(BOLD)$(RED)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Flush Redis $(NC)$(RED)$(CROSS) This will delete all data!$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -it echo-redis redis-cli -a redis_password FLUSHALL; \
		echo ""; \
		echo "$(GREEN)$(CHECK) Redis flushed$(NC)"; \
		echo ""; \
	else \
		echo ""; \
		echo "$(YELLOW)Cancelled$(NC)"; \
		echo ""; \
	fi

# =============================================================================
# DEVELOPMENT & TESTING
# =============================================================================

## setup: Initial setup
setup:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Initial Setup$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)Step 1/3:$(NC) Setting up environment files..."
	@echo ""
	@if [ ! -f .env ]; then \
		if [ -f .env.example ]; then \
			cp .env.example .env; \
			echo "  $(GREEN)$(CHECK)$(NC) Created root .env"; \
		else \
			echo "POSTGRES_USER=echo" >> .env; \
			echo "POSTGRES_PASSWORD=echo_password" >> .env; \
			echo "POSTGRES_DB=echo_db" >> .env; \
			echo "REDIS_PASSWORD=redis_password" >> .env; \
			echo "  $(GREEN)$(CHECK)$(NC) Created root .env with defaults"; \
		fi; \
	else \
		echo "  $(YELLOW)$(CROSS)$(NC) Root .env already exists"; \
	fi
	@echo ""
	@echo "$(BOLD)Step 2/3:$(NC) Setting up service .env files..."
	@echo ""
	@for service in api-gateway auth-service location-service message-service media-service; do \
		if [ ! -f services/$service/.env ]; then \
			if [ -f services/$service/.env.example ]; then \
				cp services/$service/.env.example services/$service/.env; \
				echo "  $(GREEN)$(CHECK)$(NC) Created $service/.env"; \
			else \
				echo "  $(RED)$(CROSS)$(NC) Missing $service/.env.example"; \
			fi; \
		else \
			echo "  $(YELLOW)$(CROSS)$(NC) $service/.env already exists"; \
		fi; \
	done
	@echo ""
	@echo "$(BOLD)Step 3/3:$(NC) Starting services..."
	@echo ""
	@$(MAKE) up
	@sleep 3
	@echo ""
	@echo "$(GREEN)$(CHECK) Setup complete!$(NC)"
	@echo ""
	@echo "$(BOLD)Next Steps:$(NC)"
	@echo "  $(CYAN)make health$(NC)  Verify all services"
	@echo "  $(CYAN)make logs$(NC)    View service logs"
	@echo ""

## dev: Start services in development mode with logs
dev:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Development Mode$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@$(MAKE) up
	@sleep 2
	@echo "$(BOLD)$(GREEN)Development mode active$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(MAKE) logs

## health: Check health of all services
health:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Health Check$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)API Gateway$(NC)"
	@curl -f http://localhost:8080/health >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Healthy$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)Auth Service$(NC)"
	@docker exec echo-api-gateway curl -f http://auth-service:8081/health >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Healthy$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)Location Service$(NC)"
	@curl -f http://localhost:8090/health >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Healthy$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)Message Service$(NC)"
	@curl -f http://localhost:8083/health >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Healthy$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)Media Service$(NC)"
	@docker exec echo-api-gateway curl -f http://media-service:8084/health >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Healthy$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)PostgreSQL$(NC)"
	@docker exec echo-postgres pg_isready -U echo >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Ready$(NC)" || echo "  $(RED)$(CROSS) Not ready$(NC)"
	@echo ""
	@echo "$(BOLD)Redis$(NC)"
	@docker exec echo-redis redis-cli -a redis_password PING 2>/dev/null | grep -q PONG && \
		echo "  $(GREEN)$(CHECK) Ready$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)Kafka$(NC)"
	@docker exec echo-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1 && \
		echo "  $(GREEN)$(CHECK) Ready$(NC)" || echo "  $(RED)$(CROSS) Not responding$(NC)"
	@echo ""

## test: Run tests
test:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Running Tests$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(DIM)Testing Auth Service...$(NC)"
	@cd services/auth-service && go test -v ./...
	@echo ""
	@echo "$(DIM)Testing API Gateway...$(NC)"
	@cd services/api-gateway && go test -v ./...
	@echo ""
	@echo "$(DIM)Testing Shared Packages...$(NC)"
	@cd shared/ && go test -v ./...
	@echo ""
	@echo "$(GREEN)$(CHECK) All tests completed$(NC)"
	@echo ""

## test-auth: Test auth endpoints through gateway
test-auth:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Testing Auth Endpoints$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)POST$(NC) /api/v1/auth/login"
	@echo ""
	@curl -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"test123"}' \
		2>/dev/null | jq . || echo "$(RED)$(CROSS) Request failed$(NC)"
	@echo ""

## verify-security: Verify auth service is not exposed
verify-security:
	@echo ""
	@echo "$(BOLD)$(PURPLE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Security Verification$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(BOLD)Test 1:$(NC) Direct Auth Service Access $(DIM)(should be blocked)$(NC)"
	@(curl -X POST http://localhost:8081/login -m 2 2>&1 | grep -q "Connection refused" || \
	 curl -X POST http://localhost:8081/login -m 2 2>&1 | grep -q "Failed to connect") && \
		echo "  $(GREEN)$(CHECK) Auth service is properly secured$(NC)" || \
		echo "  $(RED)$(CROSS) WARNING: Auth service is exposed!$(NC)"
	@echo ""
	@echo "$(BOLD)Test 2:$(NC) Gateway Proxy Access $(DIM)(should work)$(NC)"
	@curl -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"test123"}' \
		-s -o /dev/null -w "%{http_code}" | grep -q "200\|400\|401" && \
		echo "  $(GREEN)$(CHECK) Gateway proxy is working$(NC)" || \
		echo "  $(RED)$(CROSS) Gateway proxy failed$(NC)"
	@echo ""
	@echo "$(GREEN)$(CHECK) Security verification complete$(NC)"
	@echo ""

# =============================================================================
# UTILITIES
# =============================================================================

## generate-routes: Generate routes from registry
generate-routes:
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Generating Routes$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@go run scripts/generate-routes.go shared/routes/registry.yaml
	@echo ""
	@echo "$(GREEN)$(CHECK) Routes generated successfully$(NC)"
	@echo ""

## update-gateway-routes: Update gateway configuration
update-gateway-routes: generate-routes
	@echo ""
	@echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Updating Gateway Configuration$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@cp shared/routes/registry.yaml services/api-gateway/configs/routes.yaml
	@echo ""
	@echo "$(GREEN)$(CHECK) Gateway routes updated$(NC)"
	@echo ""
	@echo "$(BOLD)Next Step:$(NC)"
	@echo "  $(CYAN)make gateway-restart$(NC)  Restart the gateway to apply changes"
	@echo ""

format-code:
	@echo ""
	@echo "$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(BOLD)  Formatting Code$(NC)"
	@echo "$(BOLD)$(GRAY)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@find . -name '*.go' -not -path "./vendor/*" -not -path "./infra/*" | xargs gofmt -s -w
	@echo ""
	@echo "$(GREEN)$(CHECK) Code formatted successfully$(NC)"
	@echo ""