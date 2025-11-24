# =============================================================================
# DATABASE MANAGEMENT
# =============================================================================

.PHONY: db-up db-init db-seed db-connect db-clean db-reset db-migrate db-migrate-down db-migrate-status

db-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Starting PostgreSQL$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d postgres
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) PostgreSQL started$(NC)"
	@echo "  URL:  $(BRIGHT_CYAN)localhost:5432$(NC)"
	@echo "  User: $(CYAN)echo$(NC)"
	@echo "  DB:   $(CYAN)echo_db$(NC)"
	@echo ""

db-init:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Initializing Database$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x init-db.sh && ./init-db.sh --force
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Database initialized$(NC)"
	@echo ""

db-seed:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Seeding Database$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x seed-data.sh && ./seed-data.sh
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Database seeded$(NC)"
	@echo ""
	@echo "$(BOLD)Test Accounts:$(NC)"
	@echo "  $(BULLET) $(GREEN)alice@example.com$(NC)"
	@echo "  $(BULLET) $(GREEN)bob@example.com$(NC)"
	@echo "  $(BULLET) $(GREEN)charlie@example.com$(NC)"
	@echo ""

db-connect:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Connecting to PostgreSQL$(NC)"
	@echo ""
	@docker exec -it echo-postgres psql -U echo -d echo_db

db-clean:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_RED)$(CROSS) Database Clean Warning$(NC)"
	@echo "$(RED)This will delete all data!$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd infra/scripts && chmod +x clean-db.sh && ./clean-db.sh; \
		echo ""; \
		echo "$(BRIGHT_GREEN)$(CHECK) Database cleaned$(NC)"; \
		echo ""; \
	else \
		echo ""; \
		echo "$(YELLOW)$(CROSS) Cancelled$(NC)"; \
		echo ""; \
	fi

db-reset:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_RED)$(CROSS) Database Reset Warning$(NC)"
	@echo "$(RED)This will delete all data and recreate the database!$(NC)"
	@echo ""
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(YELLOW)$(ARROW) Dropping database...$(NC)"; \
		if docker exec -it echo-postgres psql -U echo -c "DROP DATABASE IF EXISTS echo_db;" 2>/dev/null; then \
			echo "$(YELLOW)$(ARROW) Creating database...$(NC)"; \
			if docker exec -it echo-postgres psql -U echo -c "CREATE DATABASE echo_db;" 2>/dev/null; then \
				echo "$(YELLOW)$(ARROW) Initializing database...$(NC)"; \
				if $(MAKE) db-init; then \
					echo ""; \
					echo "$(BRIGHT_GREEN)$(CHECK) Database reset complete$(NC)"; \
					echo ""; \
				else \
					echo ""; \
					echo "$(BRIGHT_RED)$(CROSS) Failed to initialize database$(NC)"; \
					echo ""; \
					exit 1; \
				fi; \
			else \
				echo ""; \
				echo "$(BRIGHT_RED)$(CROSS) Failed to create database$(NC)"; \
				echo ""; \
				exit 1; \
			fi; \
		else \
			echo ""; \
			echo "$(BRIGHT_RED)$(CROSS) Failed to drop database$(NC)"; \
			echo ""; \
			exit 1; \
		fi; \
	else \
		echo ""; \
		echo "$(YELLOW)$(CROSS) Cancelled$(NC)"; \
		echo ""; \
	fi

db-migrate:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_GREEN)$(STAR) Running Database Migrations$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x run-migrations.sh && ./run-migrations.sh up
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Migrations applied$(NC)"
	@echo ""

db-migrate-down:
	@echo ""
	@echo "$(BOLD)$(YELLOW)$(ARROW) Rolling Back Last Migration$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x run-migrations.sh && ./run-migrations.sh down
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Migration rolled back$(NC)"
	@echo ""

db-migrate-status:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)Migration Status$(NC)"
	@echo ""
	@cd infra/scripts && chmod +x run-migrations.sh && ./run-migrations.sh status
	@echo ""