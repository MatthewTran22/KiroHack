#!/usr/bin/env node

/**
 * Integration test script for state management with Docker containers
 * This script starts the backend services and runs state management tests
 */

const { spawn, exec } = require('child_process');
const path = require('path');

const DOCKER_COMPOSE_FILE = path.join(__dirname, '../../docker-compose.yml');
const BACKEND_HEALTH_URL = 'http://localhost:8080/health';
const MAX_RETRIES = 30;
const RETRY_DELAY = 2000;

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

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function checkBackendHealth() {
  return new Promise((resolve) => {
    const http = require('http');
    const url = require('url');
    
    const parsedUrl = url.parse(BACKEND_HEALTH_URL);
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port,
      path: parsedUrl.path,
      method: 'GET',
      timeout: 5000
    };
    
    const req = http.request(options, (res) => {
      resolve(res.statusCode === 200);
    });
    
    req.on('error', () => {
      resolve(false);
    });
    
    req.on('timeout', () => {
      req.destroy();
      resolve(false);
    });
    
    req.end();
  });
}

async function waitForBackend() {
  log('Waiting for backend to be ready...', colors.yellow);
  
  for (let i = 0; i < MAX_RETRIES; i++) {
    const isHealthy = await checkBackendHealth();
    if (isHealthy) {
      log('Backend is ready!', colors.green);
      return true;
    }
    
    log(`Attempt ${i + 1}/${MAX_RETRIES} - Backend not ready yet...`, colors.yellow);
    await sleep(RETRY_DELAY);
  }
  
  log('Backend failed to start within timeout', colors.red);
  return false;
}

async function startDockerServices() {
  log('Starting Docker services...', colors.blue);
  
  return new Promise((resolve, reject) => {
    const dockerCompose = spawn('docker-compose', [
      '-f', DOCKER_COMPOSE_FILE,
      'up', '-d',
      'app', 'mongodb', 'redis'
    ], {
      stdio: 'inherit',
      cwd: path.dirname(DOCKER_COMPOSE_FILE)
    });
    
    dockerCompose.on('close', (code) => {
      if (code === 0) {
        log('Docker services started successfully', colors.green);
        resolve();
      } else {
        reject(new Error(`Docker compose failed with code ${code}`));
      }
    });
    
    dockerCompose.on('error', (error) => {
      reject(error);
    });
  });
}

async function stopDockerServices() {
  log('Stopping Docker services...', colors.blue);
  
  return new Promise((resolve) => {
    const dockerCompose = spawn('docker-compose', [
      '-f', DOCKER_COMPOSE_FILE,
      'down'
    ], {
      stdio: 'inherit',
      cwd: path.dirname(DOCKER_COMPOSE_FILE)
    });
    
    dockerCompose.on('close', () => {
      log('Docker services stopped', colors.green);
      resolve();
    });
    
    dockerCompose.on('error', () => {
      log('Error stopping Docker services', colors.red);
      resolve();
    });
  });
}

async function runStateManagementTests() {
  log('Running state management integration tests...', colors.cyan);
  
  return new Promise((resolve, reject) => {
    const jest = spawn('npm.cmd', [
      'test', '--',
      '--testPathPatterns="stores|hooks"',
      '--testNamePattern="integration|state|management"',
      '--passWithNoTests',
      '--verbose'
    ], {
      stdio: 'inherit',
      cwd: path.join(__dirname, '..')
    });
    
    jest.on('close', (code) => {
      if (code === 0) {
        log('State management tests passed!', colors.green);
        resolve();
      } else {
        log(`State management tests failed with code ${code}`, colors.red);
        reject(new Error(`Tests failed with code ${code}`));
      }
    });
    
    jest.on('error', (error) => {
      reject(error);
    });
  });
}

async function runIntegrationTests() {
  let success = false;
  
  try {
    // Start Docker services
    await startDockerServices();
    
    // Wait for backend to be ready
    const backendReady = await waitForBackend();
    if (!backendReady) {
      throw new Error('Backend failed to start');
    }
    
    // Run state management tests
    await runStateManagementTests();
    
    success = true;
    log('All integration tests completed successfully!', colors.green);
    
  } catch (error) {
    log(`Integration tests failed: ${error.message}`, colors.red);
    process.exit(1);
  } finally {
    // Always stop Docker services
    await stopDockerServices();
  }
  
  if (success) {
    log('Integration test suite completed successfully', colors.green);
    process.exit(0);
  }
}

// Handle process termination
process.on('SIGINT', async () => {
  log('\nReceived SIGINT, cleaning up...', colors.yellow);
  await stopDockerServices();
  process.exit(1);
});

process.on('SIGTERM', async () => {
  log('\nReceived SIGTERM, cleaning up...', colors.yellow);
  await stopDockerServices();
  process.exit(1);
});

// Run the integration tests
if (require.main === module) {
  runIntegrationTests();
}

module.exports = {
  runIntegrationTests,
  startDockerServices,
  stopDockerServices,
  waitForBackend,
};