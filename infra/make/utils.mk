# =============================================================================
# UTILITIES
# =============================================================================

.PHONY: generate-routes update-gateway-routes format-code

generate-routes:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Generating Routes$(NC)"
	@echo ""
	@go run scripts/generate-routes.go shared/routes/registry.yaml
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Routes generated successfully$(NC)"
	@echo ""

update-gateway-routes: generate-routes
	@echo ""
	@echo "$(BOLD)$(BRIGHT_BLUE)$(STAR) Updating Gateway Configuration$(NC)"
	@echo ""
	@cp shared/routes/registry.yaml services/api-gateway/configs/routes.yaml
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Gateway routes updated$(NC)"
	@echo ""
	@echo "$(BOLD)Next Step:$(NC)"
	@echo "  $(BRIGHT_CYAN)make gateway-restart$(NC)  $(DIM)$(ARROW) Restart the gateway to apply changes$(NC)"
	@echo ""

format-code:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Formatting Code$(NC)"
	@echo ""
	@find . -name '*.go' -not -path "./vendor/*" -not -path "./infra/*" | xargs gofmt -s -w
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Code formatted successfully$(NC)"
	@echo ""