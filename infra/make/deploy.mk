# =============================================================================
# Laptop-side deploy targets.
#
# These run ON YOUR LAPTOP (not the VM) and wrap deploy/sync-vm.sh so you
# don't have to remember the script flags.
#
# Quick reference:
#
#   make deploy VM_IP=35.x.x.x                        # bootstrap + sync + trigger
#   make deploy VM_IP=35.x.x.x CERT=... KEY=...       # also sync TLS certs
#   make deploy-watch                                 # watch latest GHA run
#   make deploy-status DOMAIN=echo-app.net            # curl the health endpoint
#   make deploy-rollback SHA=abcdef                   # roll the VM back to a SHA
#
# Required env (or pass on the command line):
#   VM_IP    external IP of the deploy VM
#   CERT     path to TLS cert (Cloudflare Origin Cert) — optional
#   KEY      path to TLS private key — optional
# =============================================================================

VM_IP   ?=
CERT    ?=
KEY     ?=
SSH_KEY ?= $(HOME)/.ssh/echo_deploy
DOMAIN  ?= echo-app.net

.PHONY: deploy
deploy: ## End-to-end: bootstrap (if needed) + sync env/certs + trigger CI
	@if [ -z "$(VM_IP)" ]; then \
		echo "$(BRIGHT_RED)VM_IP required.$(NC) Usage: make deploy VM_IP=<ip>"; \
		exit 1; \
	fi
	@SSH_KEY="$(SSH_KEY)" bash deploy/sync-vm.sh "$(VM_IP)" $(CERT) $(KEY)

.PHONY: deploy-watch
deploy-watch: ## Tail the latest "Build & Deploy" workflow run
	@command -v gh >/dev/null || { echo "Install gh CLI: https://cli.github.com"; exit 1; }
	@gh run watch $$(gh run list --workflow=deploy.yml --limit 1 --json databaseId -q '.[0].databaseId')

.PHONY: deploy-status
deploy-status: ## Curl the public health endpoint
	@echo "$(BOLD)Caddy health:$(NC)"
	@curl -isS -m 10 "https://$(DOMAIN)/caddy-health" | head -3 || echo "  $(BRIGHT_RED)down$(NC)"
	@echo
	@echo "$(BOLD)API gateway health:$(NC)"
	@curl -isS -m 10 "https://$(DOMAIN)/api/v1/health" | head -3 || echo "  $(BRIGHT_RED)down$(NC)"

.PHONY: deploy-rollback
deploy-rollback: ## Roll the VM back to a specific SHA via workflow_dispatch
	@if [ -z "$(SHA)" ]; then \
		echo "$(BRIGHT_RED)SHA required.$(NC) Usage: make deploy-rollback SHA=<full-git-sha>"; \
		exit 1; \
	fi
	@command -v gh >/dev/null || { echo "Install gh CLI: https://cli.github.com"; exit 1; }
	gh workflow run deploy.yml --ref main \
		--field sha=$(SHA) --field profiles=caddy
	@echo "$(BRIGHT_GREEN)$(CHECK) rollback dispatched.$(NC) Watch: make deploy-watch"

.PHONY: deploy-ssh
deploy-ssh: ## Open an SSH session to the deploy VM
	@if [ -z "$(VM_IP)" ]; then \
		echo "$(BRIGHT_RED)VM_IP required.$(NC) Usage: make deploy-ssh VM_IP=<ip>"; \
		exit 1; \
	fi
	ssh -i $(SSH_KEY) deploy@$(VM_IP)

.PHONY: deploy-logs
deploy-logs: ## Tail logs of one service over SSH (SVC=foo)
	@if [ -z "$(VM_IP)" ] || [ -z "$(SVC)" ]; then \
		echo "Usage: make deploy-logs VM_IP=<ip> SVC=<service>"; \
		exit 1; \
	fi
	ssh -i $(SSH_KEY) deploy@$(VM_IP) "cd /opt/echo-backend && make vm-logs SVC=$(SVC)"
