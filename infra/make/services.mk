# =============================================================================
# MAIN SERVICES MANAGEMENT
# =============================================================================

.PHONY: up down restart build rerun logs ps status clean

up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Starting All Services$(NC)"
	@echo "$(DIM)Environment: $(ENV_NAME)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d
	@sleep 2
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Services started successfully$(NC)"
	@echo ""
	@echo "$(BOLD)Service URLs:$(NC)"
	@echo "  $(BULLET) API Gateway      $(BRIGHT_CYAN)http://localhost:8080$(NC)"
	@echo "  $(BULLET) Location Service $(BRIGHT_CYAN)http://localhost:8090$(NC)"
	@echo "  $(BULLET) WebSocket        $(BRIGHT_CYAN)ws://localhost:8083/ws$(NC)"
	@echo ""
	@echo "$(BOLD)Infrastructure:$(NC)"
	@echo "  $(BULLET) PostgreSQL  $(YELLOW)localhost:5432$(NC)"
	@echo "  $(BULLET) Redis       $(YELLOW)localhost:6379$(NC)"
	@echo "  $(BULLET) Kafka       $(YELLOW)localhost:9092$(NC)"
	@echo ""
	@echo "$(DIM)Quick commands: $(CYAN)make logs$(NC) $(DIM)|$(NC) $(CYAN)make status$(NC) $(DIM)|$(NC) $(CYAN)make health$(NC)"
	@echo ""

down:
	@echo ""
	@echo "$(BOLD)$(YELLOW)$(CROSS) Stopping All Services$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) down
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) All services stopped$(NC)"
	@echo ""

restart:
	@echo ""
	@echo "$(BOLD)$(YELLOW)$(ARROW) Restarting All Services$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) restart
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Services restarted$(NC)"
	@echo ""

build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Building All Service Images$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rebuild and Restart$(NC)"
	@echo ""
	@echo "$(DIM)$(ARROW) Step 1/3: Stopping services...$(NC)"
	@$(DOCKER_COMPOSE) down
	@echo "$(DIM)$(ARROW) Step 2/3: Building images...$(NC)"
	@$(DOCKER_COMPOSE) build --no-cache
	@echo "$(DIM)$(ARROW) Step 3/3: Starting services...$(NC)"
	@$(DOCKER_COMPOSE) up -d
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Services rebuilt and restarted$(NC)"
	@echo ""

logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f --no-log-prefix

ps:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)Running Containers$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) ps
	@echo ""

status:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)Service Status$(NC)"
	@echo ""
	@echo "$(BOLD)$(BLUE)API Gateway$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-api-gateway 2>/dev/null | grep -q true && \
		echo "  Status:  $(BRIGHT_GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(BRIGHT_RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-api-gateway 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)$(MAGENTA)Auth Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-auth-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(BRIGHT_GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(BRIGHT_RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-auth-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)$(GREEN)User Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-user-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(BRIGHT_GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(BRIGHT_RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-user-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)$(CYAN)Message Service$(NC)"
	@docker inspect -f '{{.State.Running}}' echo-message-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(BRIGHT_GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(BRIGHT_RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' echo-message-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Location Service$(NC)"
	@docker inspect -f '{{.State.Running}}' location-service 2>/dev/null | grep -q true && \
		echo "  Status:  $(BRIGHT_GREEN)$(CHECK) Running$(NC)" || echo "  Status:  $(BRIGHT_RED)$(CROSS) Stopped$(NC)"
	@echo "  Uptime:  $(CYAN)$$(docker inspect -f '{{.State.StartedAt}}' location-service 2>/dev/null | cut -d'.' -f1 || echo 'N/A')$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Infrastructure$(NC)"
	@echo "  Kafka:      $$(docker inspect -f '{{.State.Running}}' echo-kafka 2>/dev/null | grep -q true && echo '$(BRIGHT_GREEN)$(CHECK) Running$(NC)' || echo '$(BRIGHT_RED)$(CROSS) Stopped$(NC)')"
	@echo "  PostgreSQL: $$(docker inspect -f '{{.State.Running}}' echo-postgres 2>/dev/null | grep -q true && echo '$(BRIGHT_GREEN)$(CHECK) Running$(NC)' || echo '$(BRIGHT_RED)$(CROSS) Stopped$(NC)')"
	@echo "  Redis:      $$(docker inspect -f '{{.State.Running}}' echo-redis 2>/dev/null | grep -q true && echo '$(BRIGHT_GREEN)$(CHECK) Running$(NC)' || echo '$(BRIGHT_RED)$(CROSS) Stopped$(NC)')"
	@echo ""

clean:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_RED)$(CROSS) Cleanup Warning$(NC)"
	@echo "$(RED)This will delete all data including volumes!$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(YELLOW)$(ARROW) Cleaning up...$(NC)"; \
		docker-compose -f $(COMPOSE_FILE) down -v; \
		docker-compose -f $(COMPOSE_FILE) rm -f; \
		docker system prune -f; \
		docker volume prune -f; \
		docker network prune -f; \
		docker image prune -f; \
		echo ""; \
		echo "$(BRIGHT_GREEN)$(CHECK) Cleanup complete$(NC)"; \
		echo ""; \
	else \
		echo ""; \
		echo "$(YELLOW)$(CROSS) Cancelled$(NC)"; \
		echo ""; \
	fi