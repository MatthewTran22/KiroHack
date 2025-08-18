#!/usr/bin/env node

/**
 * Script to run document management integration tests with Docker containers
 * This script sets up the necessary backend services and runs comprehensive tests
 */

const { execSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// Configuration
const CONFIG = {
  DOCKER_COMPOSE_FILE: path.join(__dirname, '../../docker-compose.test.yml'),
  TEST_TIMEOUT: 120000, // 2 minutes
  BACKEND_HEALTH_CHECK_URL: 'http://localhost:8080/health',
  MAX_HEALTH_CHECK_ATTEMPTS: 30,
  HEALTH_CHECK_INTERVAL: 2000, // 2 seconds
};

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
};

function log(message, color = colors.reset) {
  console.log(`${color}${message}${colors.reset}`);
}

function logStep(step, message) {
  log(`[${step}] ${message}`, colors.blue);
}

function logSuccess(message) {
  log(`✅ ${message}`, colors.green);
}

function logError(message) {
  log(`❌ ${message}`, colors.red);
}

function logWarning(message) {
  log(`⚠️  ${message}`, colors.yellow);
}

// Check if Docker is available
function checkDockerAvailability() {
  try {
    execSync('docker --version', { stdio: 'pipe' });
    execSync('docker-compose --version', { stdio: 'pipe' });
    return true;
  } catch (error) {
    return false;
  }
}

// Wait for backend service to be healthy
async function waitForBackendHealth() {
  logStep('HEALTH', 'Waiting for backend services to be healthy...');
  
  for (let attempt = 1; attempt <= CONFIG.MAX_HEALTH_CHECK_ATTEMPTS; attempt++) {
    try {
      const response = await fetch(CONFIG.BACKEND_HEALTH_CHECK_URL);
      if (response.ok) {
        logSuccess('Backend services are healthy');
        return true;
      }
    } catch (error) {
      // Service not ready yet
    }
    
    log(`Health check attempt ${attempt}/${CONFIG.MAX_HEALTH_CHECK_ATTEMPTS}...`);
    await new Promise(resolve => setTimeout(resolve, CONFIG.HEALTH_CHECK_INTERVAL));
  }
  
  throw new Error('Backend services failed to become healthy');
}

// Start Docker services
function startDockerServices() {
  return new Promise((resolve, reject) => {
    logStep('DOCKER', 'Starting Docker services...');
    
    const dockerCompose = spawn('docker-compose', [
      '-f', CONFIG.DOCKER_COMPOSE_FILE,
      'up', '-d', '--build'
    ], {
      stdio: 'inherit',
      cwd: path.dirname(CONFIG.DOCKER_COMPOSE_FILE)
    });
    
    dockerCompose.on('close', (code) => {
      if (code === 0) {
        logSuccess('Docker services started successfully');
        resolve();
      } else {
        reject(new Error(`Docker services failed to start (exit code: ${code})`));
      }
    });
    
    dockerCompose.on('error', (error) => {
      reject(new Error(`Failed to start Docker services: ${error.message}`));
    });
  });
}

// Stop Docker services
function stopDockerServices() {
  return new Promise((resolve) => {
    logStep('CLEANUP', 'Stopping Docker services...');
    
    const dockerCompose = spawn('docker-compose', [
      '-f', CONFIG.DOCKER_COMPOSE_FILE,
      'down', '-v'
    ], {
      stdio: 'inherit',
      cwd: path.dirname(CONFIG.DOCKER_COMPOSE_FILE)
    });
    
    dockerCompose.on('close', () => {
      logSuccess('Docker services stopped');
      resolve();
    });
    
    dockerCompose.on('error', (error) => {
      logWarning(`Error stopping Docker services: ${error.message}`);
      resolve(); // Don't fail cleanup
    });
  });
}

// Run Jest tests
function runIntegrationTests() {
  return new Promise((resolve, reject) => {
    logStep('TESTS', 'Running integration tests...');
    
    const jestArgs = [
      '--testPathPattern=document-management.test.ts',
      '--testTimeout=' + CONFIG.TEST_TIMEOUT,
      '--verbose',
      '--detectOpenHandles',
      '--forceExit'
    ];
    
    const jest = spawn('npx', ['jest', ...jestArgs], {
      stdio: 'inherit',
      cwd: path.join(__dirname, '..'),
      env: {
        ...process.env,
        NODE_ENV: 'test',
        RUN_INTEGRATION_TESTS: 'true',
        TEST_API_URL: 'http://localhost:8080',
        CI: process.env.CI || 'false'
      }
    });
    
    jest.on('close', (code) => {
      if (code === 0) {
        logSuccess('Integration tests passed');
        resolve();
      } else {
        reject(new Error(`Integration tests failed (exit code: ${code})`));
      }
    });
    
    jest.on('error', (error) => {
      reject(new Error(`Failed to run integration tests: ${error.message}`));
    });
  });
}

// Create test Docker Compose file if it doesn't exist
function ensureDockerComposeFile() {
  if (!fs.existsSync(CONFIG.DOCKER_COMPOSE_FILE)) {
    logStep('SETUP', 'Creating test Docker Compose file...');
    
    const dockerComposeContent = `
version: '3.8'

services:
  # MongoDB for document storage
  mongodb:
    image: mongo:7
    container_name: test-mongodb
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password
      MONGO_INITDB_DATABASE: ai_government_consultant_test
    volumes:
      - mongodb_test_data:/data/db
      - ./scripts/mongo-init.js:/docker-entrypoint-initdb.d/mongo-init.js:ro
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Backend API service
  backend:
    build:
      context: ..
      dockerfile: Dockerfile.dev
    container_name: test-backend
    ports:
      - "8080:8080"
    environment:
      - NODE_ENV=test
      - MONGODB_URI=mongodb://admin:password@mongodb:27017/ai_government_consultant_test?authSource=admin
      - JWT_SECRET=test-jwt-secret-key-for-integration-tests
      - API_PORT=8080
      - CORS_ORIGINS=http://localhost:3000,http://localhost:3001
      - LOG_LEVEL=info
    depends_on:
      mongodb:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 10
    volumes:
      - ../internal:/app/internal
      - ../cmd:/app/cmd
      - ../pkg:/app/pkg
      - ../configs:/app/configs

volumes:
  mongodb_test_data:
`;
    
    fs.writeFileSync(CONFIG.DOCKER_COMPOSE_FILE, dockerComposeContent.trim());
    logSuccess('Docker Compose file created');
  }
}

// Main execution function
async function main() {
  try {
    log('🚀 Starting Document Management Integration Tests', colors.cyan);
    log('================================================', colors.cyan);
    
    // Check prerequisites
    if (!checkDockerAvailability()) {
      throw new Error('Docker and Docker Compose are required but not available');
    }
    
    // Ensure Docker Compose file exists
    ensureDockerComposeFile();
    
    // Start Docker services
    await startDockerServices();
    
    // Wait for services to be healthy
    await waitForBackendHealth();
    
    // Run integration tests
    await runIntegrationTests();
    
    logSuccess('All integration tests completed successfully! 🎉');
    
  } catch (error) {
    logError(`Integration tests failed: ${error.message}`);
    process.exit(1);
  } finally {
    // Always clean up Docker services
    await stopDockerServices();
  }
}

// Handle process termination
process.on('SIGINT', async () => {
  logWarning('Received SIGINT, cleaning up...');
  await stopDockerServices();
  process.exit(1);
});

process.on('SIGTERM', async () => {
  logWarning('Received SIGTERM, cleaning up...');
  await stopDockerServices();
  process.exit(1);
});

// Run if called directly
if (require.main === module) {
  main().catch((error) => {
    logError(`Unexpected error: ${error.message}`);
    process.exit(1);
  });
}

module.exports = {
  startDockerServices,
  stopDockerServices,
  waitForBackendHealth,
  runIntegrationTests,
  main
};