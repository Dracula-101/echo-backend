# =============================================================================
# KAFKA MANAGEMENT (kafka.mk)
# =============================================================================

.PHONY: kafka-up kafka-down kafka-logs kafka-restart kafka-topics kafka-create-topics

kafka-up:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Starting Kafka$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) up -d zookeeper kafka
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Kafka started$(NC)"
	@echo "  URL: $(BRIGHT_CYAN)localhost:9092$(NC)"
	@echo ""

kafka-down:
	@echo "$(YELLOW)$(ARROW) Stopping Kafka...$(NC)"
	@$(DOCKER_COMPOSE) stop kafka zookeeper
	@echo "$(BRIGHT_GREEN)$(CHECK) Kafka stopped$(NC)"

kafka-logs:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)$(STAR) Kafka Logs$(NC) $(DIM)(Press Ctrl+C to exit)$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE) logs -f kafka

kafka-restart:
	@echo "$(YELLOW)$(ARROW) Restarting Kafka...$(NC)"
	@$(DOCKER_COMPOSE) restart kafka
	@echo "$(BRIGHT_GREEN)$(CHECK) Kafka restarted$(NC)"

kafka-topics:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Kafka Topics$(NC)"
	@echo ""
	@docker exec echo-kafka kafka-topics --list --bootstrap-server localhost:9092
	@echo ""

kafka-create-topics:
	@echo ""
	@echo "$(BOLD)$(BRIGHT_YELLOW)$(STAR) Creating Kafka Topics$(NC)"
	@echo ""
	@docker exec echo-kafka kafka-topics --create --if-not-exists --bootstrap-server localhost:9092 --topic messages --partitions 3 --replication-factor 1
	@docker exec echo-kafka kafka-topics --create --if-not-exists --bootstrap-server localhost:9092 --topic notifications --partitions 3 --replication-factor 1
	@echo ""
	@echo "$(BRIGHT_GREEN)$(CHECK) Topics created$(NC)"
	@echo ""