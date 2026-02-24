# =============================================================================
# MONITORING (Prometheus, Grafana, Jaeger)
# =============================================================================

.PHONY: monitoring-up monitoring-down monitoring-logs monitoring-status

monitoring-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Starting Monitoring Stack$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) --profile monitoring up -d prometheus grafana
	@sleep 2
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Monitoring stack started$(NC)"
	@echo ""
	@echo "$(BOLD)Monitoring URLs:$(NC)"
	@echo "  $(BULLET) Prometheus $(BRIGHT_CYAN)http://localhost:9090$(NC)"
	@echo "  $(BULLET) Grafana    $(BRIGHT_CYAN)http://localhost:3000$(NC) $(DIM)(admin / admin)$(NC)"
	@echo ""

monitoring-down:
	@echo ""
	@echo "$(BOLD)$(YELLOW)$(CROSS) Stopping Monitoring Stack$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) --profile monitoring stop prometheus grafana
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Monitoring stack stopped$(NC)"
	@echo ""

monitoring-logs:
	@$(DOCKER_COMPOSE) --profile monitoring logs -f prometheus grafana

monitoring-status:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)Monitoring Status$(NC)"
	@echo ""
	@echo "  Prometheus: $$(docker inspect -f '{{.State.Running}}' echo-prometheus 2>/dev/null | grep -q true && echo '$(BRIGHT_GREEN)$(CHECK) Running$(NC)' || echo '$(BRIGHT_RED)$(CROSS) Stopped$(NC)')"
	@echo "  Grafana:    $$(docker inspect -f '{{.State.Running}}' echo-grafana 2>/dev/null | grep -q true && echo '$(BRIGHT_GREEN)$(CHECK) Running$(NC)' || echo '$(BRIGHT_RED)$(CROSS) Stopped$(NC)')"
	@echo ""
