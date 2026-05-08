# =============================================================================
# Local prod-test targets.
#
# These DON'T touch the dev stack — they bring up the real prod compose
# files on your laptop, building images locally instead of pulling from
# GHCR. Use this to catch missing env vars, broken healthchecks, or
# Dockerfile regressions BEFORE pushing to main.
#
# All targets read .env.prod (NOT .env). Copy .env.prod.example first.
# =============================================================================

PROD_COMPOSE := docker compose \
	-f infra/docker/docker-compose.prod.yml \
	-f infra/docker/docker-compose.prod-local.yml \
	--env-file .env.prod

# Profile selector. Default: minimum stack (gateway/auth/user/message/ws).
# Override on the command line:  make prod-local-up PROFILES="full"
PROFILES ?=
PROFILE_FLAGS := $(foreach p,$(PROFILES),--profile $(p))

.PHONY: prod-local-check
prod-local-check:
	@if [ ! -f .env.prod ]; then \
		echo "$(BRIGHT_RED).env.prod missing.$(NC) Run: cp .env.prod.example .env.prod && \$$EDITOR .env.prod"; \
		exit 1; \
	fi
	@grep -q "CHANGE_ME" .env.prod && { \
		echo "$(BRIGHT_YELLOW)warning: .env.prod still contains CHANGE_ME placeholders$(NC)"; \
		exit 0; \
	} || true

.PHONY: prod-local-build
prod-local-build: prod-local-check ## Build all prod images locally (no push)
	$(PROD_COMPOSE) $(PROFILE_FLAGS) build

.PHONY: prod-local-up
prod-local-up: prod-local-check ## Build images, start stack, block until healthy
	$(PROD_COMPOSE) $(PROFILE_FLAGS) up -d --build --remove-orphans --wait --wait-timeout 180
	@echo "$(BRIGHT_GREEN)$(CHECK) prod-local healthy.$(NC) Tail logs: make prod-local-logs"
	@echo "  api-gateway:  http://localhost:8080/health"
	@echo "  ws-service:   http://localhost:8086/health"

.PHONY: prod-local-down
prod-local-down: ## Stop the prod-local stack (keeps volumes)
	$(PROD_COMPOSE) $(PROFILE_FLAGS) down --remove-orphans

.PHONY: prod-local-nuke
prod-local-nuke: ## Stop AND delete all prod-local volumes (destructive)
	$(PROD_COMPOSE) $(PROFILE_FLAGS) down --remove-orphans --volumes

.PHONY: prod-local-ps
prod-local-ps: ## Show running prod-local services
	$(PROD_COMPOSE) $(PROFILE_FLAGS) ps

.PHONY: prod-local-logs
prod-local-logs: ## Tail logs (override with: make prod-local-logs SVC=ws-service)
	$(PROD_COMPOSE) $(PROFILE_FLAGS) logs -f --tail=200 $(SVC)

.PHONY: prod-local-health
prod-local-health: ## Run the same health-gate the VM deploy uses
	APP_DIR=$(PWD) PROFILES="$(PROFILES)" bash deploy/health-gate.sh "$(PROFILES)"

.PHONY: prod-local-migrate
prod-local-migrate: prod-local-check ## Re-run DB migrations against the prod-local stack
	$(PROD_COMPOSE) $(PROFILE_FLAGS) run --rm migrate

.PHONY: prod-local-psql
prod-local-psql: ## Open psql against the prod-local Postgres
	$(PROD_COMPOSE) exec postgres psql -U $$(grep ^POSTGRES_USER .env.prod | cut -d= -f2) \
	                                   -d $$(grep ^POSTGRES_DB   .env.prod | cut -d= -f2)
