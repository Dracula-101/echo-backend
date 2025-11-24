# =============================================================================
# USER SERVICE MANAGEMENT (user.mk)
# =============================================================================

.PHONY: user-up user-down user-rerun user-restart user-build user-rebuild user-logs

user-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Starting User Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d user-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) User Service started$(NC)"
	@echo ""

user-down:
	@echo "$(YELLOW)$(ARROW) Stopping User Service...$(NC)"
	@$(DOCKER_COMPOSE) stop user-service
	@echo "$(BRIGHT_GREEN)$(CHECK) User Service stopped$(NC)"

user-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Rerunning User Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop user-service
	@$(DOCKER_COMPOSE) up -d user-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) User Service rerun complete$(NC)"
	@echo ""

user-restart:
	@echo "$(YELLOW)$(ARROW) Restarting User Service...$(NC)"
	@$(DOCKER_COMPOSE) restart user-service
	@echo "$(BRIGHT_GREEN)$(CHECK) User Service restarted$(NC)"

user-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Building User Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build user-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

user-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Rebuilding User Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache user-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

user-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) User Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f user-service --no-log-prefix
