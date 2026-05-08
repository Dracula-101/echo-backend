# =============================================================================
# VM-side targets (run ON the deploy VM, after setup-vm.sh has bootstrapped).
#
# All these wrap `docker compose -f infra/docker/docker-compose.prod.yml
# --env-file .env.prod --profile caddy ...` so the operator never has to
# remember the compose invocation.
#
# Common flow on a fresh VM:
#   make vm-up           # pull latest images + bring up the stack + wait healthy
#   make vm-status       # see what's running
#   make vm-logs SVC=ws-service
#   make vm-restart SVC=api-gateway
#
# These all assume cwd = /opt/echo-backend on the VM.
# =============================================================================

VM_COMPOSE := docker compose \
	-f infra/docker/docker-compose.prod.yml \
	--env-file .env.prod \
	--profile caddy

VM_PROFILES ?=
VM_PROFILE_FLAGS := $(foreach p,$(VM_PROFILES),--profile $(p))

.PHONY: vm-check
vm-check:
	@if [ ! -f .env.prod ]; then \
		echo "$(BRIGHT_RED).env.prod missing.$(NC) Copy from .env.prod.example or scp from laptop."; \
		exit 1; \
	fi
	@if [ ! -f secrets/cert.pem ] || [ ! -f secrets/key.pem ]; then \
		echo "$(BRIGHT_YELLOW)warning:$(NC) secrets/{cert,key}.pem missing — Caddy will fail to start."; \
		echo "         scp them from your laptop, or remove the 'tls' line in Caddyfile to use Let's Encrypt."; \
	fi

.PHONY: vm-up
vm-up: vm-check ## Pull latest images and bring up the stack with --wait
	@echo "$(BOLD)$(BRIGHT_CYAN)$(ARROW) Pulling images...$(NC)"
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) pull --policy always --quiet
	@echo "$(BOLD)$(BRIGHT_CYAN)$(ARROW) Starting stack (timeout 180s)...$(NC)"
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) up -d --remove-orphans --no-build --pull never \
		--wait --wait-timeout 180
	@echo "$(BRIGHT_GREEN)$(CHECK) all services healthy$(NC)"
	@$(MAKE) vm-status

.PHONY: vm-down
vm-down: ## Stop all services (keeps volumes)
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) down --remove-orphans

.PHONY: vm-restart
vm-restart: ## Restart a service (override SVC=foo, default: all app services)
	@if [ -n "$(SVC)" ]; then \
		$(VM_COMPOSE) $(VM_PROFILE_FLAGS) restart $(SVC); \
	else \
		$(VM_COMPOSE) $(VM_PROFILE_FLAGS) restart \
			api-gateway auth-service user-service message-service ws-service; \
	fi

.PHONY: vm-status
vm-status: ## Show running services with health status
	@$(VM_COMPOSE) $(VM_PROFILE_FLAGS) ps --format "table {{.Service}}\t{{.Status}}\t{{.Image}}"

.PHONY: vm-logs
vm-logs: ## Tail logs (override with: make vm-logs SVC=ws-service)
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) logs -f --tail=200 $(SVC)

.PHONY: vm-shell
vm-shell: ## Open a shell in a service container (SVC=foo required)
	@if [ -z "$(SVC)" ]; then echo "usage: make vm-shell SVC=<service>"; exit 1; fi
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) exec $(SVC) /bin/sh

.PHONY: vm-psql
vm-psql: ## Open psql against the in-docker Postgres
	$(VM_COMPOSE) exec postgres psql \
		-U $$(grep ^POSTGRES_USER .env.prod | cut -d= -f2) \
		-d $$(grep ^POSTGRES_DB   .env.prod | cut -d= -f2)

.PHONY: vm-redis-cli
vm-redis-cli: ## Open redis-cli against the in-docker Redis
	$(VM_COMPOSE) exec redis redis-cli \
		-a $$(grep ^REDIS_PASSWORD .env.prod | cut -d= -f2)

.PHONY: vm-init-db
vm-init-db: vm-check ## Re-run db-init (idempotent; skips if schema already loaded)
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) run --rm db-init

.PHONY: vm-migrate
vm-migrate: vm-check ## Run pending migrations (database/migrations/*.sql)
	$(VM_COMPOSE) $(VM_PROFILE_FLAGS) run --rm migrate

.PHONY: vm-backup
vm-backup: ## Run a one-off Postgres backup right now
	bash infra/scripts/backup-postgres.sh

.PHONY: vm-deploy
vm-deploy: vm-check ## Run the full deploy script (checkout SHA + pull + up + wait)
	@if [ -z "$(SHA)" ]; then SHA=$$(git rev-parse HEAD); fi; \
	PROFILES="caddy" bash deploy/vm-deploy.sh "$${SHA:-$$(git rev-parse HEAD)}"

.PHONY: vm-prune
vm-prune: ## Aggressively prune dangling images and old build cache
	docker image prune -af --filter "until=72h"
	docker builder prune -f --keep-storage 1GB || true
	docker volume prune -f
