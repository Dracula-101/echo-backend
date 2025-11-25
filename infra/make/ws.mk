# =============================================================================
# WEBSOCKET SERVICE MANAGEMENT
# =============================================================================
.PHONY: ws-up ws-down ws-rerun ws-restart ws-build ws-rebuild ws-logs

ws-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Starting WebSocket Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d ws-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) WebSocket Service started$(NC)"
	@echo ""
ws-down:
	@echo "$(YELLOW)$(ARROW) Stopping WebSocket Service...$(NC)"
	@$(DOCKER_COMPOSE) stop ws-service
	@echo "$(BRIGHT_GREEN)$(CHECK) WebSocket Service stopped$(NC)"
ws-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rerunning WebSocket Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop ws-service
	@$(DOCKER_COMPOSE) up -d ws-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) WebSocket Service rerun complete$(NC)"
	@echo ""
ws-restart:
	@echo "$(YELLOW)$(ARROW) Restarting WebSocket Service...$(NC)"
	@$(DOCKER_COMPOSE) restart ws-service
	@echo "$(BRIGHT_GREEN)$(CHECK) WebSocket Service restarted$(NC)"
ws-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Building WebSocket Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build ws-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""
ws-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rebuilding WebSocket Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache ws-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""
ws-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) WebSocket Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f ws-service --no-log-prefix	
	@echo ""