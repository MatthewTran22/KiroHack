#!/usr/bin/env node

/**
 * Authentication System Test Script
 * 
 * This script demonstrates the authentication system functionality
 * and can be used to test the integration with the backend API.
 */

const { execSync } = require('child_process');
const path = require('path');

console.log('🔐 AI Government Consultant - Authentication System Test');
console.log('=' .repeat(60));

// Check if backend is running
async function checkBackend() {
  try {
    const response = await fetch('http://localhost:8080/health');
    if (response.ok) {
      console.log('✅ Backend is running at http://localhost:8080');
      return true;
    }
  } catch (error) {
    console.log('❌ Backend is not running at http://localhost:8080');
    console.log('   Start the backend with: docker-compose up -d');
    return false;
  }
}

// Run frontend tests
function runTests() {
  console.log('\n📋 Running Authentication Tests...');
  console.log('-'.repeat(40));
  
  try {
    execSync('npm test -- --testPathPatterns="auth|login" --verbose', {
      stdio: 'inherit',
      cwd: process.cwd()
    });
    console.log('\n✅ All authentication tests passed!');
  } catch (error) {
    console.log('\n❌ Some tests failed. Check the output above.');
    process.exit(1);
  }
}

// Start development server
function startDev() {
  console.log('\n🚀 Starting Development Server...');
  console.log('-'.repeat(40));
  console.log('The frontend will be available at: http://localhost:3000');
  console.log('');
  console.log('Test the authentication system:');
  console.log('1. Navigate to http://localhost:3000');
  console.log('2. You will be redirected to the login page');
  console.log('3. Try logging in with test credentials');
  console.log('4. Test MFA setup and other features');
  console.log('');
  console.log('Press Ctrl+C to stop the server');
  console.log('');
  
  try {
    execSync('npm run dev', {
      stdio: 'inherit',
      cwd: process.cwd()
    });
  } catch (error) {
    console.log('\n❌ Failed to start development server');
    process.exit(1);
  }
}

// Main execution
async function main() {
  const args = process.argv.slice(2);
  const command = args[0] || 'test';

  switch (command) {
    case 'test':
      runTests();
      break;
    
    case 'dev':
      await checkBackend();
      startDev();
      break;
    
    case 'check':
      const backendRunning = await checkBackend();
      if (backendRunning) {
        console.log('\n🔗 Integration Test Commands:');
        console.log('   INTEGRATION_TEST=true npm test -- --testPathPatterns="integration"');
      }
      break;
    
    default:
      console.log('Usage: node scripts/test-auth.js [command]');
      console.log('');
      console.log('Commands:');
      console.log('  test  - Run authentication unit tests (default)');
      console.log('  dev   - Start development server');
      console.log('  check - Check backend connectivity');
      console.log('');
      console.log('Examples:');
      console.log('  node scripts/test-auth.js test');
      console.log('  node scripts/test-auth.js dev');
      console.log('  node scripts/test-auth.js check');
  }
}

main().catch(console.error);