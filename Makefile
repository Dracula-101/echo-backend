# =============================================================================
# ECHO BACKEND - MAIN MAKEFILE
# =============================================================================
# Modular Makefile structure with clean separation of concerns
# =============================================================================

.PHONY: help

# =============================================================================
# CONFIGURATION
# =============================================================================

# Environment Selection (dev/prod)
ENV ?= dev

# Docker Compose files
COMPOSE_FILE_BASE := infra/docker/docker-compose.yml
COMPOSE_FILE_DEV := infra/docker/docker-compose.dev.yml
COMPOSE_FILE_PROD := infra/docker/docker-compose.prod.yml

# Build compose command with base file and environment-specific overlay
ifeq ($(ENV),prod)
    COMPOSE_FILES := -f $(COMPOSE_FILE_BASE) -f $(COMPOSE_FILE_PROD)
    ENV_NAME := Production
else
    COMPOSE_FILES := -f $(COMPOSE_FILE_BASE) -f $(COMPOSE_FILE_DEV)
    ENV_NAME := Development
endif

# Compose command alias
DOCKER_COMPOSE := docker-compose $(COMPOSE_FILES)

# Export for submakefiles
export COMPOSE_FILES
export DOCKER_COMPOSE
export ENV
export ENV_NAME

# Default target
.DEFAULT_GOAL := help

# =============================================================================
# COLORS & STYLING
# =============================================================================

# Text styles
BOLD := \033[1m
DIM := \033[2m
ITALIC := \033[3m
UNDERLINE := \033[4m

# Colors
BLACK := \033[0;30m
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
MAGENTA := \033[0;35m
CYAN := \033[0;36m
WHITE := \033[0;37m

# Bright colors
BRIGHT_RED := \033[1;31m
BRIGHT_GREEN := \033[1;32m
BRIGHT_YELLOW := \033[1;33m
BRIGHT_BLUE := \033[1;34m
BRIGHT_MAGENTA := \033[1;35m
BRIGHT_CYAN := \033[1;36m

# Reset
NC := \033[0m

# Symbols
CHECK := ✓
CROSS := ✗
ARROW := →
BULLET := •
STAR := ★

# Export for submakefiles
export BOLD DIM ITALIC UNDERLINE
export BLACK RED GREEN YELLOW BLUE MAGENTA CYAN WHITE
export BRIGHT_RED BRIGHT_GREEN BRIGHT_YELLOW BRIGHT_BLUE BRIGHT_MAGENTA BRIGHT_CYAN
export NC
export CHECK CROSS ARROW BULLET STAR

# =============================================================================
# INCLUDE MODULAR MAKEFILES
# =============================================================================

include infra/make/help.mk
include infra/make/services.mk
include infra/make/gateway.mk
include infra/make/auth.mk
include infra/make/user.mk
include infra/make/message.mk
include infra/make/location.mk
include infra/make/media.mk
include infra/make/presence.mk
include infra/make/kafka.mk
include infra/make/database.mk
include infra/make/redis.mk
include infra/make/dev.mk
include infra/make/utils.mk