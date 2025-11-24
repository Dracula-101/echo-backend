# =============================================================================
# DEVELOPMENT & TESTING
# =============================================================================

.PHONY: setup dev health test test-auth verify-security

setup:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Initial Setup$(NC)"
	@echo ""
	@echo "$(DIM)$(ARROW) Step 1/3: Setting up environment files...$(NC)"
	@if [ ! -f .env ]; then \
		if [ -f .env.example ]; then \
			cp .env.example .env; \
			echo "  $(BRIGHT_GREEN)$(CHECK)$(NC) Created root .env"; \
		else \
			echo "POSTGRES_USER=echo" >> .env; \
			echo "POSTGRES_PASSWORD=echo_password" >> .env; \
			echo "POSTGRES_DB=echo_db" >> .env; \
			echo "REDIS_PASSWORD=redis_password" >> .env; \
			echo "  $(BRIGHT_GREEN)$(CHECK)$(NC) Created root .env with defaults"; \
		fi; \
	else \
		echo "  $(YELLOW)$(BULLET)$(NC) Root .env already exists"; \
	fi
	@echo ""
	@echo "$(DIM)$(ARROW) Step 2/3: Setting up service .env files...$(NC)"
	@for service in api-gateway auth-service location-service message-service media-service; do \
		if [ ! -f services/$$service/.env ]; then \
			if [ -f services/$$service/.env.example ]; then \
				cp services/$$service/.env.example services/$$service/.env; \
				echo "  $(BRIGHT_GREEN)$(CHECK)$(NC) Created $$service/.env"; \
			else \
				echo "  $(BRIGHT_RED)$(CROSS)$(NC) Missing $$service/.env.example"; \
			fi; \
		else \
			echo "  $(YELLOW)$(BULLET)$(NC) $$service/.env already exists"; \
		fi; \
	done
	@echo ""
	@echo "$(DIM)$(ARROW) Step 3/3: Starting services...$(NC)"
	@$(MAKE) up
	@sleep 3
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Setup complete!$(NC)"
	@echo ""
	@echo "$(BOLD)Next Steps:$(NC)"
	@echo "  $(BRIGHT_CYAN)make health$(NC)  $(DIM)$(ARROW) Verify all services$(NC)"
	@echo "  $(BRIGHT_CYAN)make logs$(NC)    $(DIM)$(ARROW) View service logs$(NC)"
	@echo ""

dev:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Development Mode$(NC)"
	@echo "$(DIM)Environment: $(ENV_NAME)$(NC)"
	@echo ""
	@$(MAKE) up
	@sleep 2
	@echo "$(BOLD)$(BRIGHT_GREEN)$(CHECK) Development mode active$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(MAKE) logs

health:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Health Check$(NC)"
	@echo ""
	@echo "$(BOLD)$(BLUE)API Gateway$(NC)"
	@curl -f http://localhost:8080/health >/dev/null 2>&1 && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Healthy$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)$(MAGENTA)Auth Service$(NC)"
	@docker exec echo-api-gateway curl -f http://auth-service:8081/health >/dev/null 2>&1 && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Healthy$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Location Service$(NC)"
	@curl -f http://localhost:8090/health >/dev/null 2>&1 && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Healthy$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)$(CYAN)Message Service$(NC)"
	@curl -f http://localhost:8083/health >/dev/null 2>&1 && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Healthy$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)$(GREEN)PostgreSQL$(NC)"
	@docker exec echo-postgres pg_isready -U echo >/dev/null 2>&1 && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Ready$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not ready$(NC)"
	@echo ""
	@echo "$(BOLD)$(RED)Redis$(NC)"
	@docker exec echo-redis redis-cli -a redis_password PING 2>/dev/null | grep -q PONG && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Ready$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not responding$(NC)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Kafka$(NC)"
	@docker exec echo-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 >/dev/null 2>&1 && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Ready$(NC)" || echo "  $(BRIGHT_RED)$(CROSS) Not responding$(NC)"
	@echo ""

test:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Running Tests$(NC)"
	@echo ""
	@echo "$(DIM)$(ARROW) Testing Auth Service...$(NC)"
	@cd services/auth-service && go test -v ./...
	@echo ""
	@echo "$(DIM)$(ARROW) Testing API Gateway...$(NC)"
	@cd services/api-gateway && go test -v ./...
	@echo ""
	@echo "$(DIM)$(ARROW) Testing Shared Packages...$(NC)"
	@cd shared/ && go test -v ./...
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) All tests completed$(NC)"
	@echo ""

test-auth:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Testing Auth Endpoints$(NC)"
	@echo ""
	@echo "$(BOLD)POST$(NC) /api/v1/auth/login"
	@echo ""
	@curl -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"test123"}' \
		2>/dev/null | jq . || echo "$(BRIGHT_RED)$(CROSS) Request failed$(NC)"
	@echo ""

verify-security:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_MAGENTA)$(STAR) Security Verification$(NC)"
	@echo ""
	@echo "$(BOLD)Test 1:$(NC) Direct Auth Service Access $(DIM)(should be blocked)$(NC)"
	@(curl -X POST http://localhost:8081/login -m 2 2>&1 | grep -q "Connection refused" || \
	 curl -X POST http://localhost:8081/login -m 2 2>&1 | grep -q "Failed to connect") && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Auth service is properly secured$(NC)" || \
		echo "  $(BRIGHT_RED)$(CROSS) WARNING: Auth service is exposed!$(NC)"
	@echo ""
	@echo "$(BOLD)Test 2:$(NC) Gateway Proxy Access $(DIM)(should work)$(NC)"
	@curl -X POST http://localhost:8080/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"test123"}' \
		-s -o /dev/null -w "%{http_code}" | grep -q "200\|400\|401" && \
		echo "  $(BRIGHT_GREEN)$(CHECK) Gateway proxy is working$(NC)" || \
		echo "  $(BRIGHT_RED)$(CROSS) Gateway proxy failed$(NC)"
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Security verification complete$(NC)"
	@echo ""