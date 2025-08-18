#!/bin/bash

# Integration test script using Docker containers
set -e

echo "🚀 Starting integration tests with Docker containers..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    print_status "Cleaning up test containers..."
    docker-compose -f docker-compose.test.yml down -v --remove-orphans || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    print_error "docker-compose is not installed. Please install docker-compose and try again."
    exit 1
fi

print_status "Stopping any existing test containers..."
docker-compose -f docker-compose.test.yml down -v --remove-orphans || true

print_status "Building test images..."
docker-compose -f docker-compose.test.yml build

print_status "Starting test dependencies (MongoDB and Redis)..."
docker-compose -f docker-compose.test.yml up -d test-mongodb test-redis

print_status "Waiting for services to be healthy..."
timeout=60
counter=0
while [ $counter -lt $timeout ]; do
    if docker-compose -f docker-compose.test.yml ps | grep -q "healthy"; then
        print_status "Services are healthy!"
        break
    fi
    sleep 2
    counter=$((counter + 2))
    echo -n "."
done

if [ $counter -ge $timeout ]; then
    print_error "Services failed to become healthy within $timeout seconds"
    docker-compose -f docker-compose.test.yml logs
    exit 1
fi

print_status "Running integration tests..."
if docker-compose -f docker-compose.test.yml run --rm test-runner; then
    print_status "✅ All integration tests passed!"
    exit 0
else
    print_error "❌ Integration tests failed!"
    print_status "Showing container logs for debugging..."
    docker-compose -f docker-compose.test.yml logs
    exit 1
fi