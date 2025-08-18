package integration

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestDockerContainerConnectivity tests connectivity to Docker containers
func TestDockerContainerConnectivity(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping Docker connectivity tests in short mode")
	}

	t.Run("MongoDB Container Connectivity", func(t *testing.T) {
		// Connect to test MongoDB container
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		mongoURI := "mongodb://testadmin:testpassword@localhost:27018/ai_government_consultant_test?authSource=admin"
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		require.NoError(t, err, "Failed to connect to test MongoDB")
		defer client.Disconnect(ctx)

		// Test the connection
		err = client.Ping(ctx, nil)
		assert.NoError(t, err, "Failed to ping test MongoDB")

		// Test database operations
		db := client.Database("ai_government_consultant_test")
		collection := db.Collection("test_collection")

		// Insert a test document
		testDoc := map[string]interface{}{
			"test_field": "test_value",
			"timestamp":  time.Now(),
		}
		result, err := collection.InsertOne(ctx, testDoc)
		assert.NoError(t, err, "Failed to insert test document")
		assert.NotNil(t, result.InsertedID, "Insert result should have an ID")

		// Clean up test document
		_, err = collection.DeleteOne(ctx, map[string]interface{}{"_id": result.InsertedID})
		assert.NoError(t, err, "Failed to clean up test document")
	})

	t.Run("Redis Container Connectivity", func(t *testing.T) {
		// Connect to test Redis container
		redisClient := redis.NewClient(&redis.Options{
			Addr:     "localhost:6380",
			Password: "testpassword",
			DB:       0,
		})
		defer redisClient.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test the connection
		pong, err := redisClient.Ping(ctx).Result()
		assert.NoError(t, err, "Failed to ping test Redis")
		assert.Equal(t, "PONG", pong, "Redis ping should return PONG")

		// Test Redis operations
		testKey := "test_key"
		testValue := "test_value"

		// Set a test value
		err = redisClient.Set(ctx, testKey, testValue, time.Minute).Err()
		assert.NoError(t, err, "Failed to set test value in Redis")

		// Get the test value
		retrievedValue, err := redisClient.Get(ctx, testKey).Result()
		assert.NoError(t, err, "Failed to get test value from Redis")
		assert.Equal(t, testValue, retrievedValue, "Retrieved value should match set value")

		// Clean up test key
		err = redisClient.Del(ctx, testKey).Err()
		assert.NoError(t, err, "Failed to clean up test key")
	})
}

// TestDockerContainerHealth tests that containers are healthy
func TestDockerContainerHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker health tests in short mode")
	}

	t.Run("Container Health Status", func(t *testing.T) {
		// This test verifies that the containers started successfully
		// The actual health checks are done by Docker Compose
		
		// Test MongoDB connectivity with a simple operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mongoURI := "mongodb://testadmin:testpassword@localhost:27018/ai_government_consultant_test?authSource=admin"
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		if err != nil {
			t.Logf("MongoDB connection failed: %v", err)
			t.Skip("MongoDB container not available, skipping health test")
		}
		defer client.Disconnect(ctx)

		err = client.Ping(ctx, nil)
		assert.NoError(t, err, "MongoDB container should be healthy")

		// Test Redis connectivity
		redisClient := redis.NewClient(&redis.Options{
			Addr:     "localhost:6380",
			Password: "testpassword",
			DB:       0,
		})
		defer redisClient.Close()

		_, err = redisClient.Ping(ctx).Result()
		assert.NoError(t, err, "Redis container should be healthy")
	})
}