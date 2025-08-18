#!/usr/bin/env node

const { execSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const PROJECT_ROOT = path.resolve(__dirname, '../..');
const FRONTEND_ROOT = path.resolve(__dirname, '..');

console.log('🚀 Starting API Integration Tests with Docker');

// Check if Docker is available
try {
  execSync('docker --version', { stdio: 'ignore' });
  console.log('✅ Docker is available');
} catch (error) {
  console.error('❌ Docker is not available. Please install Docker to run integration tests.');
  process.exit(1);
}

// Check if docker-compose files exist
const composeFiles = [
  path.join(PROJECT_ROOT, 'docker-compose.yml'),
  path.join(PROJECT_ROOT, 'docker-compose.test.yml'),
];

for (const file of composeFiles) {
  if (!fs.existsSync(file)) {
    console.error(`❌ Docker compose file not found: ${file}`);
    process.exit(1);
  }
}

console.log('✅ Docker compose files found');

// Function to run command and stream output
function runCommand(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    console.log(`Running: ${command} ${args.join(' ')}`);
    
    const child = spawn(command, args, {
      stdio: 'inherit',
      cwd: options.cwd || PROJECT_ROOT,
      ...options,
    });

    child.on('close', (code) => {
      if (code === 0) {
        resolve();
      } else {
        reject(new Error(`Command failed with exit code ${code}`));
      }
    });

    child.on('error', (error) => {
      reject(error);
    });
  });
}

// Function to wait for service to be ready
async function waitForService(url, maxAttempts = 30, delay = 2000) {
  console.log(`⏳ Waiting for service at ${url} to be ready...`);
  
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        console.log(`✅ Service at ${url} is ready`);
        return;
      }
    } catch (error) {
      // Service not ready yet
    }

    if (attempt === maxAttempts) {
      throw new Error(`Service at ${url} did not become ready within ${maxAttempts * delay / 1000} seconds`);
    }

    console.log(`⏳ Attempt ${attempt}/${maxAttempts} - waiting ${delay/1000}s...`);
    await new Promise(resolve => setTimeout(resolve, delay));
  }
}

async function main() {
  let servicesStarted = false;

  try {
    // Step 1: Build the application
    console.log('\n📦 Building the application...');
    await runCommand('docker-compose', [
      '-f', 'docker-compose.yml',
      '-f', 'docker-compose.test.yml',
      'build'
    ]);

    // Step 2: Start the services
    console.log('\n🚀 Starting services...');
    await runCommand('docker-compose', [
      '-f', 'docker-compose.yml',
      '-f', 'docker-compose.test.yml',
      'up', '-d'
    ]);
    servicesStarted = true;

    // Step 3: Wait for services to be ready
    console.log('\n⏳ Waiting for services to be ready...');
    
    // Wait for backend API
    await waitForService('http://localhost:8080/health');
    
    // Wait for database (if accessible)
    try {
      await waitForService('http://localhost:27017', 10, 1000);
    } catch (error) {
      console.log('⚠️  Database health check failed, but continuing...');
    }

    // Step 4: Run database migrations/setup if needed
    console.log('\n🗄️  Setting up test database...');
    try {
      await runCommand('docker-compose', [
        '-f', 'docker-compose.yml',
        '-f', 'docker-compose.test.yml',
        'exec', '-T', 'backend',
        'go', 'run', 'cmd/server/main.go', '--migrate'
      ]);
    } catch (error) {
      console.log('⚠️  Database setup failed, but continuing...');
    }

    // Step 5: Create test user
    console.log('\n👤 Creating test user...');
    try {
      await runCommand('docker-compose', [
        '-f', 'docker-compose.yml',
        '-f', 'docker-compose.test.yml',
        'exec', '-T', 'backend',
        'go', 'run', 'scripts/create-test-user.go'
      ]);
    } catch (error) {
      console.log('⚠️  Test user creation failed, but continuing...');
    }

    // Step 6: Run the integration tests
    console.log('\n🧪 Running API integration tests...');
    await runCommand('npm', ['test', '--', '--testPathPattern=api-integration.test.ts', '--runInBand'], {
      cwd: FRONTEND_ROOT,
      env: {
        ...process.env,
        NODE_ENV: 'test',
        API_BASE_URL: 'http://localhost:8080',
        INTEGRATION_TEST: 'true',
      },
    });

    console.log('\n✅ All integration tests passed!');

  } catch (error) {
    console.error('\n❌ Integration tests failed:', error.message);
    process.exit(1);
  } finally {
    if (servicesStarted) {
      // Step 7: Cleanup - Stop and remove containers
      console.log('\n🧹 Cleaning up services...');
      try {
        await runCommand('docker-compose', [
          '-f', 'docker-compose.yml',
          '-f', 'docker-compose.test.yml',
          'down', '-v'
        ]);
        console.log('✅ Services cleaned up');
      } catch (error) {
        console.error('⚠️  Cleanup failed:', error.message);
      }
    }
  }
}

// Handle process termination
process.on('SIGINT', async () => {
  console.log('\n🛑 Received SIGINT, cleaning up...');
  try {
    await runCommand('docker-compose', [
      '-f', 'docker-compose.yml',
      '-f', 'docker-compose.test.yml',
      'down', '-v'
    ]);
  } catch (error) {
    console.error('Cleanup failed:', error.message);
  }
  process.exit(0);
});

process.on('SIGTERM', async () => {
  console.log('\n🛑 Received SIGTERM, cleaning up...');
  try {
    await runCommand('docker-compose', [
      '-f', 'docker-compose.yml',
      '-f', 'docker-compose.test.yml',
      'down', '-v'
    ]);
  } catch (error) {
    console.error('Cleanup failed:', error.message);
  }
  process.exit(0);
});

// Add fetch polyfill for Node.js
if (typeof fetch === 'undefined') {
  global.fetch = require('node-fetch');
}

main().catch((error) => {
  console.error('Script failed:', error);
  process.exit(1);
});