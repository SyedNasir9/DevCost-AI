#!/bin/bash
# DevCost AI Setup Script
# Run this script to set up the development environment

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    DevCost AI Setup                          ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is installed
check_docker() {
    echo "Checking Docker installation..."
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}✗ Docker is not installed${NC}"
        echo "  Please install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker is installed${NC}"

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo -e "${RED}✗ Docker Compose is not installed${NC}"
        echo "  Please install Docker Compose: https://docs.docker.com/compose/install/"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker Compose is installed${NC}"
}

# Check if Docker daemon is running
check_docker_running() {
    echo "Checking if Docker is running..."
    if ! docker info &> /dev/null; then
        echo -e "${RED}✗ Docker daemon is not running${NC}"
        echo "  Please start Docker Desktop or the Docker daemon"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker is running${NC}"
}

# Check port availability
check_ports() {
    echo "Checking port availability..."
    
    ports=("3000:Frontend" "8080:Backend" "5432:PostgreSQL")
    all_available=true
    
    for port_info in "${ports[@]}"; do
        port="${port_info%%:*}"
        name="${port_info#*:}"
        
        if lsof -i:"$port" &> /dev/null || netstat -tuln 2>/dev/null | grep -q ":$port "; then
            echo -e "${YELLOW}⚠ Port $port ($name) is in use${NC}"
            all_available=false
        else
            echo -e "${GREEN}✓ Port $port ($name) is available${NC}"
        fi
    done
    
    if [ "$all_available" = false ]; then
        echo ""
        echo -e "${YELLOW}Some ports are in use. You may need to stop other services or modify .env${NC}"
    fi
}

# Create .env file
setup_env() {
    echo ""
    echo "Setting up environment configuration..."
    
    if [ -f .env ]; then
        echo -e "${YELLOW}⚠ .env file already exists${NC}"
        read -p "  Overwrite? (y/N): " overwrite
        if [ "$overwrite" != "y" ] && [ "$overwrite" != "Y" ]; then
            echo "  Keeping existing .env file"
            return
        fi
    fi
    
    cp .env.example .env
    echo -e "${GREEN}✓ Created .env file from .env.example${NC}"
}

# Print configuration instructions
print_instructions() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                  Configuration Options                        ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "The system is configured to run in DEMO_MODE by default."
    echo "This means it will use mock data without requiring AWS credentials."
    echo ""
    echo -e "${YELLOW}To use real AWS data:${NC}"
    echo "  1. Edit .env file"
    echo "  2. Set DEMO_MODE=false"
    echo "  3. Add your AWS credentials:"
    echo "     AWS_ACCESS_KEY_ID=your-key"
    echo "     AWS_SECRET_ACCESS_KEY=your-secret"
    echo ""
    echo -e "${YELLOW}Optional configurations:${NC}"
    echo "  • Slack integration: Add SLACK_BOT_TOKEN and SLACK_SIGNING_SECRET"
    echo "  • Email alerts: Add SMTP_* settings"
    echo ""
}

# Start services
start_services() {
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    Starting Services                          ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    
    # Use docker compose (v2) or docker-compose (v1)
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    else
        COMPOSE_CMD="docker-compose"
    fi
    
    echo "Building and starting containers..."
    $COMPOSE_CMD up -d --build
    
    echo ""
    echo "Waiting for services to be healthy..."
    sleep 10
    
    # Check service health
    echo ""
    $COMPOSE_CMD ps
}

# Print success message
print_success() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                    Setup Complete! 🎉                         ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo -e "${GREEN}DevCost AI is now running!${NC}"
    echo ""
    echo "  Frontend:  http://localhost:3000"
    echo "  Backend:   http://localhost:8080"
    echo "  API Docs:  http://localhost:8080/swagger/index.html"
    echo "  Health:    http://localhost:8080/health"
    echo ""
    echo "Useful commands:"
    echo "  View logs:     docker compose logs -f"
    echo "  Stop:          docker compose down"
    echo "  Restart:       docker compose restart"
    echo "  Rebuild:       docker compose up -d --build"
    echo ""
}

# Main execution
main() {
    check_docker
    check_docker_running
    check_ports
    setup_env
    print_instructions
    
    read -p "Start DevCost AI now? (Y/n): " start_now
    if [ "$start_now" != "n" ] && [ "$start_now" != "N" ]; then
        start_services
        print_success
    else
        echo ""
        echo "To start later, run: docker compose up -d --build"
    fi
}

main
