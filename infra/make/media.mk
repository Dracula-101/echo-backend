
# =============================================================================
# MEDIA SERVICE MANAGEMENT (media.mk)
# =============================================================================

.PHONY: media-up media-down media-rerun media-restart media-build media-rebuild media-logs

media-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Starting Media Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d media-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Media Service started$(NC)"
	@echo ""

media-down:
	@echo "$(YELLOW)$(ARROW) Stopping Media Service...$(NC)"
	@$(DOCKER_COMPOSE) stop media-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Media Service stopped$(NC)"

media-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rerunning Media Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop media-service
	@$(DOCKER_COMPOSE) up -d media-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Media Service rerun complete$(NC)"
	@echo ""

media-restart:
	@echo "$(YELLOW)$(ARROW) Restarting Media Service...$(NC)"
	@$(DOCKER_COMPOSE) restart media-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Media Service restarted$(NC)"

media-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Building Media Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build media-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

media-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rebuilding Media Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache media-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

media-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Media Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f media-service --no-log-prefix
