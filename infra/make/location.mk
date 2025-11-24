
# =============================================================================
# LOCATION SERVICE MANAGEMENT (location.mk)
# =============================================================================

.PHONY: location-up location-down location-rerun location-restart location-build location-rebuild location-logs

location-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Starting Location Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d location-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Location Service started$(NC)"
	@echo ""

location-down:
	@echo "$(YELLOW)$(ARROW) Stopping Location Service...$(NC)"
	@$(DOCKER_COMPOSE) stop location-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Location Service stopped$(NC)"

location-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Rerunning Location Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop location-service
	@$(DOCKER_COMPOSE) up -d location-service --build --force-recreate
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Location Service rerun complete$(NC)"
	@echo ""

location-restart:
	@echo "$(YELLOW)$(ARROW) Restarting Location Service...$(NC)"
	@$(DOCKER_COMPOSE) restart location-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Location Service restarted$(NC)"

location-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Building Location Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build location-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

location-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Rebuilding Location Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache location-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

location-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Location Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f location-service --no-log-prefix
