# =============================================================================
# API GATEWAY MANAGEMENT
# =============================================================================

.PHONY: gateway-up gateway-down gateway-rerun gateway-restart gateway-build gateway-rebuild gateway-logs

gateway-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Starting API Gateway$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d api-gateway
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) API Gateway started$(NC)"
	@echo "  URL: $(BRIGHT_CYAN)http://localhost:8080$(NC)"
	@echo ""

gateway-down:
	@echo ""
	@echo "$(YELLOW)$(ARROW) Stopping API Gateway...$(NC)"
	@$(DOCKER_COMPOSE) stop api-gateway
	@echo "$(BRIGHT_GREEN)$(CHECK) API Gateway stopped$(NC)"
	@echo ""

gateway-rerun:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Rerunning API Gateway$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) stop api-gateway
	@$(DOCKER_COMPOSE) up -d api-gateway
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) API Gateway rerun complete$(NC)"
	@echo ""

gateway-restart:
	@echo ""
	@echo "$(YELLOW)$(ARROW) Restarting API Gateway...$(NC)"
	@$(DOCKER_COMPOSE) restart api-gateway
	@echo "$(BRIGHT_GREEN)$(CHECK) API Gateway restarted$(NC)"
	@echo ""

gateway-build:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Building API Gateway$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build api-gateway
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Build complete$(NC)"
	@echo ""

gateway-rebuild:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Rebuilding API Gateway$(NC) $(DIM)(no cache)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) build --no-cache api-gateway
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Rebuild complete$(NC)"
	@echo ""

gateway-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) API Gateway Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f api-gateway --no-log-prefix