package auth

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/models"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestConfig holds test configuration
type TestConfig struct {
	MongoURI  string
	RedisAddr string
}

// SetupTestMongoDB sets up a test MongoDB connection
func SetupTestMongoDB(t *testing.T) (*mongo.Collection, func()) {
	// Use in-memory MongoDB for testing or skip if not available
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return nil, nil
	}

	// Test connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return nil, nil
	}

	// Use a test database
	db := client.Database("test_auth_" + primitive.NewObjectID().Hex())
	collection := db.Collection("users")

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return collection, cleanup
}

// SetupTestRedis sets up a test Redis connection
func SetupTestRedis(t *testing.T) (*redis.Client, func()) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use a different DB for testing
	})

	// Test connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		t.Skip("Redis not available for testing")
		return nil, nil
	}

	cleanup := func() {
		client.FlushDB(context.Background())
		client.Close()
	}

	return client, cleanup
}

// CreateTestAuthService creates a test auth service with test dependencies
func CreateTestAuthService(t *testing.T) (*AuthService, func()) {
	userCollection, mongoCleanup := SetupTestMongoDB(t)
	if userCollection == nil {
		return nil, nil
	}

	redisClient, redisCleanup := SetupTestRedis(t)
	if redisClient == nil {
		mongoCleanup()
		return nil, nil
	}

	jwtConfig := JWTConfig{
		AccessSecret:  "test-access-secret",
		RefreshSecret: "test-refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    24 * time.Hour,
		SessionTTL:    30 * time.Minute,
		BlacklistTTL:  24 * time.Hour,
		Issuer:        "test-issuer",
	}

	authService := NewAuthService(userCollection, redisClient, jwtConfig)

	cleanup := func() {
		redisCleanup()
		mongoCleanup()
	}

	return authService, cleanup
}

// CreateTestUser creates a test user for testing
func CreateTestUser() *models.User {
	return &models.User{
		ID:                primitive.NewObjectID(),
		Email:             "test@example.com",
		Name:              "Test User",
		Department:        "Test Department",
		Role:              models.UserRoleAnalyst,
		SecurityClearance: models.SecurityClearanceSecret,
		IsActive:          true,
		MFAEnabled:        false,
		Permissions: []models.Permission{
			{
				Resource: "documents",
				Actions:  []string{"read", "write"},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestUserWithMFA creates a test user with MFA enabled
func CreateTestUserWithMFA() *models.User {
	user := CreateTestUser()
	user.MFAEnabled = true
	user.MFASecret = "JBSWY3DPEHPK3PXP" // Test secret
	return user
}

// CreateTestCredentials creates test credentials
func CreateTestCredentials() AuthCredentials {
	return AuthCredentials{
		Email:    "test@example.com",
		Password: "TestPassword123!",
	}
}

// CreateTestCredentialsWithMFA creates test credentials with MFA code
func CreateTestCredentialsWithMFA(mfaCode string) AuthCredentials {
	creds := CreateTestCredentials()
	creds.MFACode = mfaCode
	return creds
}

// WaitForRedis waits for Redis operations to complete
func WaitForRedis() {
	time.Sleep(10 * time.Millisecond)
}

// AssertNoError asserts that an error is nil
func AssertNoError(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatalf("%s: %v", message, err)
	}
}

// AssertError asserts that an error is not nil
func AssertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Fatalf("%s: expected error but got none", message)
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	if !condition {
		t.Errorf("%s: expected true but got false", message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	if condition {
		t.Errorf("%s: expected false but got true", message)
	}
}
