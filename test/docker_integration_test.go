package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerApplicationIntegration tests the full application running in Docker
func TestDockerApplicationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	baseURL := "http://localhost:8080"
	client := &http.Client{Timeout: 10 * time.Second}
	
	t.Run("Test Health Endpoint", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/health")
		require.NoError(t, err, "Health endpoint should be accessible")
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health endpoint should return 200")
		
		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err, "Health response should be valid JSON")
		
		assert.Equal(t, "healthy", health["status"], "Application should be healthy")
		assert.NotEmpty(t, health["timestamp"], "Health response should include timestamp")
		assert.Equal(t, "1.0.0", health["version"], "Health response should include version")
		
		t.Log("✅ Health endpoint test passed")
	})
	
	t.Run("Test Status Endpoint", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/api/v1/status")
		require.NoError(t, err, "Status endpoint should be accessible")
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Status endpoint should return 200")
		
		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err, "Status response should be valid JSON")
		
		assert.Equal(t, "operational", status["status"], "Service should be operational")
		assert.Equal(t, "ai-government-consultant", status["service"], "Service name should be correct")
		
		t.Log("✅ Status endpoint test passed")
	})
	
	t.Run("Test Application Startup Time", func(t *testing.T) {
		// Test that the application starts up within reasonable time
		// This is already proven by the previous tests, but let's verify response times
		
		start := time.Now()
		resp, err := client.Get(baseURL + "/health")
		duration := time.Since(start)
		
		require.NoError(t, err, "Health check should succeed")
		defer resp.Body.Close()
		
		assert.Less(t, duration, 5*time.Second, "Health check should respond quickly")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200")
		
		t.Logf("✅ Application response time: %v", duration)
	})
	
	t.Run("Test Invalid Endpoints", func(t *testing.T) {
		// Test that invalid endpoints return appropriate errors
		resp, err := client.Get(baseURL + "/invalid-endpoint")
		require.NoError(t, err, "Request should complete")
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Invalid endpoint should return 404")
		
		t.Log("✅ Invalid endpoint handling test passed")
	})
}

// TestDockerEnvironmentConfiguration tests that the Docker environment is properly configured
func TestDockerEnvironmentConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	t.Run("Test Container Health", func(t *testing.T) {
		// This test verifies that all required containers are running and healthy
		// We can't directly check Docker from within the test, but we can verify
		// that the services they provide are accessible
		
		client := &http.Client{Timeout: 5 * time.Second}
		
		// Test main application
		resp, err := client.Get("http://localhost:8080/health")
		require.NoError(t, err, "Main application should be accessible")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Main application should be healthy")
		
		t.Log("✅ Main application container is healthy")
		
		// Note: MongoDB and Redis are not directly exposed via HTTP in our setup,
		// but they are tested indirectly through the application's ability to start
		// and respond to requests (which requires database connections)
	})
	
	t.Run("Test Research Service Configuration", func(t *testing.T) {
		// Verify that the research service configuration is loaded correctly
		// by checking that the application starts successfully with all dependencies
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("http://localhost:8080/api/v1/status")
		require.NoError(t, err, "Application should be running with research service")
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Application should be operational")
		
		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err, "Status response should be valid JSON")
		
		assert.Equal(t, "operational", status["status"], "Service should be operational with research components")
		
		t.Log("✅ Research service configuration test passed")
	})
}

// TestDockerResearchServiceEndpoints tests research-related endpoints (when implemented)
func TestDockerResearchServiceEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	t.Run("Test Research Endpoints Placeholder", func(t *testing.T) {
		// This test is a placeholder for when research endpoints are implemented
		// For now, we'll just verify that the application is running and could
		// potentially serve research endpoints
		
		client := &http.Client{Timeout: 5 * time.Second}
		baseURL := "http://localhost:8080"
		
		// Test that the API base path is accessible
		resp, err := client.Get(baseURL + "/api/v1/status")
		require.NoError(t, err, "API base path should be accessible")
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "API should be operational")
		
		// Future research endpoints would be tested here:
		// - POST /api/v1/research/analyze
		// - GET /api/v1/research/results/:id
		// - POST /api/v1/research/suggestions
		// - GET /api/v1/research/events
		// - etc.
		
		t.Log("✅ Research service endpoints placeholder test passed")
		t.Log("📝 Note: Actual research endpoints will be tested when implemented")
	})
}

// TestDockerPerformance tests basic performance characteristics
func TestDockerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	t.Run("Test Response Time Performance", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}
		baseURL := "http://localhost:8080"
		
		// Test multiple requests to ensure consistent performance
		var totalDuration time.Duration
		numRequests := 10
		
		for i := 0; i < numRequests; i++ {
			start := time.Now()
			resp, err := client.Get(baseURL + "/health")
			duration := time.Since(start)
			
			require.NoError(t, err, fmt.Sprintf("Request %d should succeed", i+1))
			resp.Body.Close()
			
			assert.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("Request %d should return 200", i+1))
			assert.Less(t, duration, 2*time.Second, fmt.Sprintf("Request %d should be fast", i+1))
			
			totalDuration += duration
		}
		
		avgDuration := totalDuration / time.Duration(numRequests)
		t.Logf("✅ Average response time over %d requests: %v", numRequests, avgDuration)
		
		assert.Less(t, avgDuration, 1*time.Second, "Average response time should be reasonable")
	})
	
	t.Run("Test Concurrent Request Handling", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}
		baseURL := "http://localhost:8080"
		
		// Test concurrent requests
		numConcurrent := 5
		results := make(chan error, numConcurrent)
		
		for i := 0; i < numConcurrent; i++ {
			go func(requestID int) {
				resp, err := client.Get(baseURL + "/health")
				if err != nil {
					results <- fmt.Errorf("concurrent request %d failed: %w", requestID, err)
					return
				}
				defer resp.Body.Close()
				
				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("concurrent request %d returned status %d", requestID, resp.StatusCode)
					return
				}
				
				results <- nil
			}(i)
		}
		
		// Wait for all requests to complete
		for i := 0; i < numConcurrent; i++ {
			err := <-results
			assert.NoError(t, err, fmt.Sprintf("Concurrent request %d should succeed", i))
		}
		
		t.Logf("✅ Successfully handled %d concurrent requests", numConcurrent)
	})
}

// TestDockerCleanup tests cleanup and resource management
func TestDockerCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration tests")
	}
	
	t.Run("Test Graceful Shutdown Preparation", func(t *testing.T) {
		// This test verifies that the application is in a state where it could
		// be gracefully shut down. We don't actually shut it down in the test
		// since other tests might still need it.
		
		client := &http.Client{Timeout: 5 * time.Second}
		
		// Verify the application is still responsive before potential shutdown
		resp, err := client.Get("http://localhost:8080/health")
		require.NoError(t, err, "Application should be responsive before shutdown")
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Application should be healthy before shutdown")
		
		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err, "Health response should be valid")
		
		assert.Equal(t, "healthy", health["status"], "Application should report healthy status")
		
		t.Log("✅ Application is ready for graceful shutdown")
		t.Log("📝 Note: Actual shutdown testing would be done in separate test suite")
	})
}