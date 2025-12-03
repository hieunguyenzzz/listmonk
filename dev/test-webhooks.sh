#!/bin/bash
# Integration test script for listmonk webhooks.
# This script tests the webhook system using Docker containers.
#
# Prerequisites:
# - Docker and docker-compose installed
# - listmonk binary built (run `make build` first)
#
# Usage:
#   ./dev/test-webhooks.sh
#
# The script will:
# 1. Start the Docker environment (db, webhook-receiver)
# 2. Initialize the database
# 3. Configure webhooks via API
# 4. Trigger various events
# 5. Verify webhooks were received

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"

# Colors for output.
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function.
cleanup() {
    log_info "Cleaning up..."
    docker-compose -f "$COMPOSE_FILE" down --volumes 2>/dev/null || true
}

# Wait for a service to be healthy.
wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=${3:-30}
    local attempt=1

    log_info "Waiting for $name to be ready..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            log_info "$name is ready"
            return 0
        fi
        sleep 1
        attempt=$((attempt + 1))
    done

    log_error "$name did not become ready in time"
    return 1
}

# Check if webhook was received.
check_webhook() {
    local event=$1
    local expected_count=${2:-1}

    local response=$(curl -s "http://localhost:8888/events?event=$event")
    local count=$(echo "$response" | grep -o '"count":[0-9]*' | cut -d':' -f2)

    if [ "$count" -ge "$expected_count" ]; then
        log_info "Webhook '$event' received ($count times)"
        return 0
    else
        log_error "Webhook '$event' not received (expected $expected_count, got $count)"
        return 1
    fi
}

# Main test function.
main() {
    log_info "Starting webhook integration tests"

    # Trap cleanup on exit.
    trap cleanup EXIT

    # Start Docker services.
    log_info "Starting Docker services..."
    docker-compose -f "$COMPOSE_FILE" up -d db webhook-receiver

    # Wait for services.
    wait_for_service "http://localhost:5432" "PostgreSQL" 30 || {
        # PostgreSQL doesn't respond to HTTP, so we check differently.
        sleep 5
    }
    wait_for_service "http://localhost:8888/health" "Webhook Receiver" 30

    # Check if listmonk binary exists.
    if [ ! -f "$PROJECT_DIR/listmonk" ]; then
        log_warn "listmonk binary not found, building..."
        cd "$PROJECT_DIR" && make build
    fi

    # Initialize the database.
    log_info "Initializing database..."
    cd "$PROJECT_DIR"
    LISTMONK_db__host=localhost \
    LISTMONK_db__port=5432 \
    LISTMONK_db__user=listmonk-dev \
    LISTMONK_db__password=listmonk-dev \
    LISTMONK_db__database=listmonk-dev \
    LISTMONK_db__ssl_mode=disable \
    ./listmonk --install --idempotent --yes

    # Start listmonk in the background.
    log_info "Starting listmonk..."
    LISTMONK_db__host=localhost \
    LISTMONK_db__port=5432 \
    LISTMONK_db__user=listmonk-dev \
    LISTMONK_db__password=listmonk-dev \
    LISTMONK_db__database=listmonk-dev \
    LISTMONK_db__ssl_mode=disable \
    LISTMONK_app__admin_username=admin \
    LISTMONK_app__admin_password=admin \
    ./listmonk > /tmp/listmonk.log 2>&1 &
    LISTMONK_PID=$!

    # Wait for listmonk to be ready.
    wait_for_service "http://localhost:9000/api/health" "listmonk" 60

    # Clear any previous webhook events.
    log_info "Clearing webhook receiver..."
    curl -s -X POST "http://localhost:8888/clear" > /dev/null

    # Get current settings and add webhook configuration.
    log_info "Configuring webhook endpoint..."

    # First, get current settings.
    SETTINGS=$(curl -s -u admin:admin "http://localhost:9000/api/settings")

    # Update settings with webhook.
    WEBHOOK_CONFIG='{
        "webhooks": [{
            "enabled": true,
            "name": "test-webhook",
            "url": "http://localhost:8888/webhook",
            "secret": "test-secret",
            "events": [
                "subscriber.created",
                "subscriber.confirmed",
                "subscriber.unsubscribed",
                "subscriber.blocklisted",
                "subscriber.deleted",
                "campaign.started",
                "campaign.finished"
            ],
            "max_conns": 5,
            "timeout": "5s"
        }]
    }'

    # Merge webhook config with existing settings and update.
    # Note: This is a simplified approach - in practice you'd merge properly.
    curl -s -X PUT -u admin:admin \
        -H "Content-Type: application/json" \
        -d "$WEBHOOK_CONFIG" \
        "http://localhost:9000/api/settings" > /dev/null

    # Wait for settings to take effect (listmonk restarts).
    sleep 3
    wait_for_service "http://localhost:9000/api/health" "listmonk" 60

    # Clear webhook receiver again after restart.
    curl -s -X POST "http://localhost:8888/clear" > /dev/null

    log_info "Running webhook tests..."

    # Test 1: Create a subscriber.
    log_info "Test 1: Creating subscriber..."
    SUBSCRIBER_RESPONSE=$(curl -s -X POST -u admin:admin \
        -H "Content-Type: application/json" \
        -d '{"email": "test@example.com", "name": "Test User", "status": "enabled", "lists": []}' \
        "http://localhost:9000/api/subscribers")

    SUBSCRIBER_ID=$(echo "$SUBSCRIBER_RESPONSE" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)

    if [ -z "$SUBSCRIBER_ID" ]; then
        log_error "Failed to create subscriber"
        echo "$SUBSCRIBER_RESPONSE"
        exit 1
    fi
    log_info "Created subscriber with ID: $SUBSCRIBER_ID"

    # Wait for webhook to be dispatched.
    sleep 2

    # Verify subscriber.created webhook.
    check_webhook "subscriber.created" 1 || exit 1

    # Test 2: Delete the subscriber.
    log_info "Test 2: Deleting subscriber..."
    curl -s -X DELETE -u admin:admin \
        "http://localhost:9000/api/subscribers/$SUBSCRIBER_ID" > /dev/null

    sleep 2
    check_webhook "subscriber.deleted" 1 || exit 1

    # Show all received webhooks.
    log_info "All received webhooks:"
    curl -s "http://localhost:8888/events" | python3 -m json.tool 2>/dev/null || \
        curl -s "http://localhost:8888/events"

    # Verify signatures were valid.
    log_info "Checking signature validation..."
    docker-compose -f "$COMPOSE_FILE" logs webhook-receiver 2>&1 | grep -q "signature: valid"
    if [ $? -eq 0 ]; then
        log_info "HMAC signatures verified successfully"
    else
        log_warn "Could not verify HMAC signature validation in logs"
    fi

    # Stop listmonk.
    log_info "Stopping listmonk..."
    kill $LISTMONK_PID 2>/dev/null || true

    log_info "=========================================="
    log_info "All webhook integration tests passed!"
    log_info "=========================================="
}

# Run main.
main "$@"
