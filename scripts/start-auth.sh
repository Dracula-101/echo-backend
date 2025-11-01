#!/bin/bash

# Echo Backend - Quick Start Script for Auth Service
# This script will help you get the auth service running quickly

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Echo Backend - Auth Service Setup   ║${NC}"
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo ""

# Get script directory and navigate to project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo -e "${BLUE}Project Root: ${PROJECT_ROOT}${NC}"
echo ""

# Check if Docker is running
echo -e "${YELLOW}[1/5] Checking Docker...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}✗ Docker is not running. Please start Docker Desktop.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"
echo ""

# Check if .env exists
echo -e "${YELLOW}[2/5] Checking environment configuration...${NC}"
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env file from template...${NC}"
    cp .env.auth.example .env
    echo -e "${GREEN}✓ Created .env file${NC}"
else
    echo -e "${GREEN}✓ .env file exists${NC}"
fi
echo ""

# Start services
echo -e "${YELLOW}[3/5] Starting services (this may take a minute)...${NC}"
docker-compose -f services/auth-service/docker-compose.auth.yml up -d

echo -e "${GREEN}✓ Services started${NC}"
echo ""

# Wait for services to be healthy
echo -e "${YELLOW}[4/5] Waiting for services to be ready...${NC}"
echo -e "${YELLOW}This may take up to 40 seconds...${NC}"

for i in {1..40}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Auth service is ready!${NC}"
        break
    fi
    echo -n "."
    sleep 1
done
echo ""

# Initialize database
echo -e "${YELLOW}[5/5] Initializing database...${NC}"
cd infra/scripts
chmod +x init-db.sh
./init-db.sh
cd ../..
echo -e "${GREEN}✓ Database initialized${NC}"
echo ""

# Ask about test data
echo -e "${YELLOW}Would you like to seed the database with test data? (y/N)${NC}"
read -r response
if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo -e "${YELLOW}Seeding database...${NC}"
    cd infra/scripts
    chmod +x seed-data.sh
    ./seed-data.sh
    cd ../..
    echo ""
fi

# Success message
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     🎉 Setup Complete! 🎉            ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Auth Service:${NC}       http://localhost:8080"
echo -e "${BLUE}Health Check:${NC}      http://localhost:8080/health"
echo -e "${BLUE}PostgreSQL:${NC}        localhost:5432 (user: echo, db: echo_db)"
echo -e "${BLUE}Redis:${NC}             localhost:6379"
echo ""
echo -e "${YELLOW}Quick Test:${NC}"
echo -e "  curl http://localhost:8080/health"
echo ""
echo -e "${YELLOW}View Logs:${NC}"
echo -e "  docker-compose -f services/auth-service/docker-compose.auth.yml logs -f auth-service"
echo ""
echo -e "${YELLOW}Using Makefile:${NC}"
echo -e "  make help          - Show all available commands"
echo -e "  make auth-logs     - View auth service logs"
echo -e "  make status        - Check service status"
echo -e "  make db-seed       - Seed database with test data"
echo ""
echo -e "${YELLOW}Stop Services:${NC}"
echo -e "  docker-compose -f services/auth-service/docker-compose.auth.yml down"
echo -e "  # or"
echo -e "  make auth-down"
echo ""

# Test the service
echo -e "${YELLOW}Testing auth service...${NC}"
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
if echo "$HEALTH_RESPONSE" | grep -q "ok"; then
    echo -e "${GREEN}✓ Auth service is responding correctly${NC}"
    echo -e "${BLUE}Response: ${HEALTH_RESPONSE}${NC}"
else
    echo -e "${RED}✗ Auth service might not be fully ready yet${NC}"
    echo -e "${YELLOW}Wait a few more seconds and try: curl http://localhost:8080/health${NC}"
fi
echo ""

echo -e "${GREEN}Happy coding! 🚀${NC}"
