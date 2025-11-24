# =============================================================================
# REDIS MANAGEMENT (redis.mk)
# =============================================================================

.PHONY: redis-connect redis-flush

redis-connect:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_RED)$(STAR) Connecting to Redis$(NC)"
	@echo ""
	@docker exec -it echo-redis redis-cli -a redis_password

redis-flush:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_RED)$(CROSS) Flush Redis Warning$(NC)"
	@echo "$(RED)This will delete all cached data!$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -it echo-redis redis-cli -a redis_password FLUSHALL; \
		echo ""; \
		echo "$(BRIGHT_GREEN)$(CHECK) Redis flushed$(NC)"; \
		echo ""; \
	else \
		echo ""; \
		echo "$(YELLOW)$(CROSS) Cancelled$(NC)"; \
		echo ""; \
	fi
