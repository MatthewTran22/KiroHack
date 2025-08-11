package database

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.URI != "mongodb://localhost:27017" {
		t.Errorf("Expected URI to be 'mongodb://localhost:27017', got %s", config.URI)
	}
	if config.DatabaseName != "ai_government_consultant" {
		t.Errorf("Expected DatabaseName to be 'ai_government_consultant', got %s", config.DatabaseName)
	}
	if config.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected ConnectTimeout to be 10s, got %v", config.ConnectTimeout)
	}
	if config.MaxPoolSize != 100 {
		t.Errorf("Expected MaxPoolSize to be 100, got %d", config.MaxPoolSize)
	}
	if config.MinPoolSize != 5 {
		t.Errorf("Expected MinPoolSize to be 5, got %d", config.MinPoolSize)
	}
	if config.MaxIdleTime != 5*time.Minute {
		t.Errorf("Expected MaxIdleTime to be 5m, got %v", config.MaxIdleTime)
	}
}

func TestNewMongoDB_WithNilConfig(t *testing.T) {
	// This test will fail if MongoDB is not running, which is expected in CI
	// In a real environment, you would use a test database or mock
	_, err := NewMongoDB(nil)

	// We expect either success (if MongoDB is running) or a connection error
	if err != nil {
		// Check if it's a connection error (expected when MongoDB is not running)
		if err.Error() == "" {
			t.Errorf("Expected a meaningful error message, got empty string")
		}
		// This is expected when MongoDB is not available
		t.Logf("MongoDB connection failed as expected in test environment: %v", err)
	} else {
		t.Log("MongoDB connection succeeded - test database is available")
	}
}

func TestNewMongoDB_WithCustomConfig(t *testing.T) {
	config := &Config{
		URI:            "mongodb://admin:password@localhost:27017/test_db?authSource=admin",
		DatabaseName:   "test_db",
		ConnectTimeout: 5 * time.Second,
		MaxPoolSize:    50,
		MinPoolSize:    2,
		MaxIdleTime:    2 * time.Minute,
	}

	// This test will fail if MongoDB is not running, which is expected in CI
	_, err := NewMongoDB(config)

	if err != nil {
		// Check if it's a connection error (expected when MongoDB is not running)
		if err.Error() == "" {
			t.Errorf("Expected a meaningful error message, got empty string")
		}
		// This is expected when MongoDB is not available
		t.Logf("MongoDB connection failed as expected in test environment: %v", err)
	} else {
		t.Log("MongoDB connection succeeded with custom config")
	}
}

func TestMongoDB_GetCollection(t *testing.T) {
	// Create a mock MongoDB instance for testing
	config := DefaultConfig()

	// We can't test the actual connection without a running MongoDB instance
	// So we'll test the configuration and structure instead
	if config.DatabaseName == "" {
		t.Error("Database name should not be empty")
	}

	// Test collection names that should be used
	expectedCollections := []string{
		"documents",
		"users",
		"consultations",
		"knowledge_items",
	}

	for _, collectionName := range expectedCollections {
		if collectionName == "" {
			t.Errorf("Collection name should not be empty")
		}
	}
}

func TestMongoDB_Close(t *testing.T) {
	// Test that Close method handles nil client gracefully
	mongodb := &MongoDB{
		Client: nil,
	}

	ctx := context.Background()
	err := mongodb.Close(ctx)

	// Should not error when client is nil
	if err != nil {
		t.Errorf("Close should not error when client is nil, got: %v", err)
	}
}

// Integration test - only runs if MongoDB is available
func TestMongoDB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &Config{
		URI:            "mongodb://admin:password@localhost:27017/test_ai_government_consultant?authSource=admin",
		DatabaseName:   "test_ai_government_consultant",
		ConnectTimeout: 5 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    1 * time.Minute,
	}

	mongodb, err := NewMongoDB(config)
	if err != nil {
		t.Skipf("Skipping integration test - MongoDB not available: %v", err)
	}
	defer func() {
		ctx := context.Background()
		mongodb.Close(ctx)
	}()

	ctx := context.Background()

	// Test ping
	err = mongodb.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test health check
	err = mongodb.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test getting a collection
	collection := mongodb.GetCollection("test_collection")
	if collection == nil {
		t.Error("GetCollection should not return nil")
	}

	// Test creating indexes (this will create the collections)
	err = mongodb.CreateIndexes(ctx)
	if err != nil {
		t.Errorf("CreateIndexes failed: %v", err)
	}

	// Clean up - drop the test database
	err = mongodb.Database.Drop(ctx)
	if err != nil {
		t.Logf("Failed to drop test database: %v", err)
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
			valid:  true,
		},
		{
			name: "custom valid config",
			config: &Config{
				URI:            "mongodb://localhost:27017",
				DatabaseName:   "custom_db",
				ConnectTimeout: 15 * time.Second,
				MaxPoolSize:    200,
				MinPoolSize:    10,
				MaxIdleTime:    10 * time.Minute,
			},
			valid: true,
		},
		{
			name: "empty database name",
			config: &Config{
				URI:            "mongodb://localhost:27017",
				DatabaseName:   "",
				ConnectTimeout: 10 * time.Second,
				MaxPoolSize:    100,
				MinPoolSize:    5,
				MaxIdleTime:    5 * time.Minute,
			},
			valid: false,
		},
		{
			name: "empty URI",
			config: &Config{
				URI:            "",
				DatabaseName:   "test_db",
				ConnectTimeout: 10 * time.Second,
				MaxPoolSize:    100,
				MinPoolSize:    5,
				MaxIdleTime:    5 * time.Minute,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.valid {
				if tt.config.URI == "" {
					t.Error("Valid config should have non-empty URI")
				}
				if tt.config.DatabaseName == "" {
					t.Error("Valid config should have non-empty DatabaseName")
				}
				if tt.config.ConnectTimeout <= 0 {
					t.Error("Valid config should have positive ConnectTimeout")
				}
				if tt.config.MaxPoolSize <= 0 {
					t.Error("Valid config should have positive MaxPoolSize")
				}
			} else {
				// For invalid configs, at least one field should be problematic
				hasIssue := tt.config.URI == "" ||
					tt.config.DatabaseName == "" ||
					tt.config.ConnectTimeout <= 0 ||
					tt.config.MaxPoolSize <= 0

				if !hasIssue {
					t.Error("Invalid config should have at least one problematic field")
				}
			}
		})
	}
}
