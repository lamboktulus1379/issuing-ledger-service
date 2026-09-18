#!/bin/bash

# Go Project Service Manager
# Companion script for easy service management

DEPLOY_SCRIPT="$(dirname "$0")/deploy-nginx.sh"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_help() {
    echo "Go Project Service Manager"
    echo
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  start      Deploy and start all services"
    echo "  stop       Stop all services"
    echo "  restart    Restart all services"
    echo "  status     Show service status"
    echo "  logs       Show Go application logs"
    echo "  test       Run deployment tests"
    echo "  help       Show this help message"
    echo
    echo "Examples:"
    echo "  $0 start     # Deploy and start everything"
    echo "  $0 status    # Check if services are running"
    echo "  $0 logs      # Watch application logs"
    echo
}

main() {
    case "${1:-help}" in
        "start"|"deploy")
            echo -e "${BLUE}Starting Go Project deployment...${NC}"
            APP_PORT=10010 "$DEPLOY_SCRIPT"
            ;;
        "stop")
            echo -e "${YELLOW}Stopping services...${NC}"
            "$DEPLOY_SCRIPT" stop
            ;;
        "restart")
            echo -e "${YELLOW}Restarting services...${NC}"
            "$DEPLOY_SCRIPT" restart
            ;;
        "status")
            "$DEPLOY_SCRIPT" status
            ;;
        "logs")
            "$DEPLOY_SCRIPT" logs
            ;;
        "test")
            "$DEPLOY_SCRIPT" test
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            echo "Unknown command: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

main "$@"
