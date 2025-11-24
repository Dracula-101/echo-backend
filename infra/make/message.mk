
# =============================================================================
# MESSAGE SERVICE MANAGEMENT (message.mk)
# =============================================================================

.PHONY: message-up message-down message-rerun message-restart message-build message-rebuild message-logs

message-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Starting Message Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d message-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Message Service started$(NC)"
	@echo "  REST API:   $(BRIGHT_CYAN)http://localhost:8083$(NC)"
	@echo "  WebSocket:  $(BRIGHT_CYAN)ws://localhost:8083/ws$(NC)"
	@echo ""

message-down:
	@echo "$(YELLOW)$(ARROW) Stopping Message Service...$(NC)"
	@$(DOCKER_COMPOSE) stop message-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Message Service stopped$(NC)"

message-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Rerunning Message Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop message-service
	@$(DOCKER_COMPOSE) up -d message-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Message Service rerun complete$(NC)"
	@echo ""

message-restart:
	@echo "$(YELLOW)$(ARROW) Restarting Message Service...$(NC)"
	@$(DOCKER_COMPOSE) restart message-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Message Service restarted$(NC)"

message-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Building Message Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build message-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

message-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Rebuilding Message Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache message-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

message-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Message Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f message-service --no-log-prefix
