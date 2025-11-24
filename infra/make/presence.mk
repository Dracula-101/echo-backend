
# =============================================================================
# PRESENCE SERVICE MANAGEMENT (presence.mk)
# =============================================================================

.PHONY: presence-up presence-down presence-rerun presence-restart presence-build presence-rebuild presence-logs

presence-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Starting Presence Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d presence-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Presence Service started$(NC)"
	@echo ""

presence-down:
	@echo "$(YELLOW)$(ARROW) Stopping Presence Service...$(NC)"
	@$(DOCKER_COMPOSE) stop presence-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Presence Service stopped$(NC)"

presence-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Rerunning Presence Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop presence-service
	@$(DOCKER_COMPOSE) up -d presence-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Presence Service rerun complete$(NC)"
	@echo ""

presence-restart:
	@echo "$(YELLOW)$(ARROW) Restarting Presence Service...$(NC)"
	@$(DOCKER_COMPOSE) restart presence-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Presence Service restarted$(NC)"

presence-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Building Presence Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build presence-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

presence-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Rebuilding Presence Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache presence-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

presence-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Presence Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f presence-service --no-log-prefix