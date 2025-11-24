# =============================================================================
# AUTH SERVICE MANAGEMENT
# =============================================================================

.PHONY: auth-up auth-down auth-rerun auth-restart auth-build auth-rebuild auth-logs

auth-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Starting Auth Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d auth-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Auth Service started$(NC)"
	@echo ""

auth-down:
	@echo "$(YELLOW)$(ARROW) Stopping Auth Service...$(NC)"
	@$(DOCKER_COMPOSE) stop auth-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Auth Service stopped$(NC)"

auth-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rerunning Auth Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop auth-service
	@$(DOCKER_COMPOSE) up -d auth-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Auth Service rerun complete$(NC)"
	@echo ""

auth-restart:
	@echo "$(YELLOW)$(ARROW) Restarting Auth Service...$(NC)"
	@$(DOCKER_COMPOSE) restart auth-service
	@echo "$(BRIGHT_GREEN)$(CHECK) Auth Service restarted$(NC)"

auth-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Building Auth Service$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build auth-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

auth-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rebuilding Auth Service$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache auth-service
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

auth-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Auth Service Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f auth-service --no-log-prefix