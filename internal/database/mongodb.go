package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDB represents a MongoDB database connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	config   *Config
}

// Config holds the MongoDB configuration
type Config struct {
	URI            string
	DatabaseName   string
	ConnectTimeout time.Duration
	MaxPoolSize    uint64
	MinPoolSize    uint64
	MaxIdleTime    time.Duration
}

// DefaultConfig returns a default MongoDB configuration
func DefaultConfig() *Config {
	return &Config{
		URI:            "mongodb://localhost:27017",
		DatabaseName:   "ai_government_consultant",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    100,
		MinPoolSize:    5,
		MaxIdleTime:    5 * time.Minute,
	}
}

// NewMongoDB creates a new MongoDB connection with the given configuration
func NewMongoDB(config *Config) (*MongoDB, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Set client options
	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetConnectTimeout(config.ConnectTimeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize).
		SetMaxConnIdleTime(config.MaxIdleTime)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.DatabaseName)

	return &MongoDB{
		Client:   client,
		Database: database,
		config:   config,
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}

// Ping pings the MongoDB server to check connectivity
func (m *MongoDB) Ping(ctx context.Context) error {
	return m.Client.Ping(ctx, readpref.Primary())
}

// GetCollection returns a MongoDB collection
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// CreateIndexes creates the necessary indexes for the application
func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	// Create indexes for documents collection
	if err := m.createDocumentIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create document indexes: %w", err)
	}

	// Create indexes for users collection
	if err := m.createUserIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	// Create indexes for consultations collection
	if err := m.createConsultationIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create consultation indexes: %w", err)
	}

	// Create indexes for knowledge_items collection
	if err := m.createKnowledgeIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create knowledge indexes: %w", err)
	}

	// Create indexes for research collections
	if err := m.createResearchIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create research indexes: %w", err)
	}

	return nil
}

// createDocumentIndexes creates indexes for the documents collection
func (m *MongoDB) createDocumentIndexes(ctx context.Context) error {
	collection := m.GetCollection("documents")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"name", 1}},
		},
		{
			Keys: bson.D{{"uploaded_by", 1}},
		},
		{
			Keys: bson.D{{"uploaded_at", -1}},
		},
		{
			Keys: bson.D{{"processing_status", 1}},
		},
		{
			Keys: bson.D{{"metadata.category", 1}},
		},
		{
			Keys: bson.D{{"metadata.tags", 1}},
		},
		{
			Keys: bson.D{{"classification.level", 1}},
		},
		// Text index for full-text search
		{
			Keys: bson.D{{"name", "text"}, {"content", "text"}},
		},
		// Vector search index for embeddings (MongoDB Atlas Vector Search)
		// Note: This would need to be created through MongoDB Atlas UI or specific vector search commands
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createUserIndexes creates indexes for the users collection
func (m *MongoDB) createUserIndexes(ctx context.Context) error {
	collection := m.GetCollection("users")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{"email", 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{"department", 1}},
		},
		{
			Keys: bson.D{{"role", 1}},
		},
		{
			Keys: bson.D{{"security_clearance", 1}},
		},
		{
			Keys: bson.D{{"is_active", 1}},
		},
		{
			Keys: bson.D{{"last_login", -1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createConsultationIndexes creates indexes for the consultations collection
func (m *MongoDB) createConsultationIndexes(ctx context.Context) error {
	collection := m.GetCollection("consultations")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"user_id", 1}},
		},
		{
			Keys: bson.D{{"type", 1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
		{
			Keys: bson.D{{"tags", 1}},
		},
		// Compound index for user queries
		{
			Keys: bson.D{{"user_id", 1}, {"created_at", -1}},
		},
		// Text index for searching consultation queries and responses
		{
			Keys: bson.D{{"query", "text"}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createKnowledgeIndexes creates indexes for the knowledge_items collection
func (m *MongoDB) createKnowledgeIndexes(ctx context.Context) error {
	collection := m.GetCollection("knowledge_items")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"type", 1}},
		},
		{
			Keys: bson.D{{"category", 1}},
		},
		{
			Keys: bson.D{{"tags", 1}},
		},
		{
			Keys: bson.D{{"keywords", 1}},
		},
		{
			Keys: bson.D{{"created_by", 1}},
		},
		{
			Keys: bson.D{{"created_at", -1}},
		},
		{
			Keys: bson.D{{"updated_at", -1}},
		},
		{
			Keys: bson.D{{"is_active", 1}},
		},
		{
			Keys: bson.D{{"confidence", -1}},
		},
		{
			Keys: bson.D{{"validation.is_validated", 1}},
		},
		{
			Keys: bson.D{{"validation.expires_at", 1}},
		},
		// Compound indexes for common queries
		{
			Keys: bson.D{{"type", 1}, {"is_active", 1}},
		},
		{
			Keys: bson.D{{"category", 1}, {"confidence", -1}},
		},
		// Text index for full-text search
		{
			Keys: bson.D{{"title", "text"}, {"content", "text"}, {"summary", "text"}},
		},
		// Vector search index for embeddings (MongoDB Atlas Vector Search)
		// Note: This would need to be created through MongoDB Atlas UI or specific vector search commands
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createResearchIndexes creates indexes for research-related collections
func (m *MongoDB) createResearchIndexes(ctx context.Context) error {
	// Research results indexes
	researchResultsCollection := m.GetCollection("research_results")
	researchIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"document_id", 1}}},
		{Keys: bson.D{{"status", 1}}},
		{Keys: bson.D{{"generated_at", -1}}},
		{Keys: bson.D{{"confidence", -1}}},
		{Keys: bson.D{{"research_query", "text"}}},
	}
	
	_, err := researchResultsCollection.Indexes().CreateMany(ctx, researchIndexes)
	if err != nil {
		return fmt.Errorf("failed to create research results indexes: %w", err)
	}
	
	// Policy suggestions indexes
	policySuggestionsCollection := m.GetCollection("policy_suggestions")
	policyIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"category", 1}}},
		{Keys: bson.D{{"priority", 1}}},
		{Keys: bson.D{{"status", 1}}},
		{Keys: bson.D{{"created_at", -1}}},
		{Keys: bson.D{{"confidence", -1}}},
		{Keys: bson.D{{"created_by", 1}}},
		{Keys: bson.D{{"tags", 1}}},
		{Keys: bson.D{{"title", "text"}, {"description", "text"}}},
	}
	
	_, err = policySuggestionsCollection.Indexes().CreateMany(ctx, policyIndexes)
	if err != nil {
		return fmt.Errorf("failed to create policy suggestions indexes: %w", err)
	}
	
	// Current events indexes
	currentEventsCollection := m.GetCollection("current_events")
	eventIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"category", 1}}},
		{Keys: bson.D{{"tags", 1}}},
		{Keys: bson.D{{"published_at", -1}}},
		{Keys: bson.D{{"relevance", -1}}},
		{Keys: bson.D{{"source", 1}}},
		{Keys: bson.D{{"language", 1}}},
		{Keys: bson.D{{"url", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"title", "text"}, {"description", "text"}, {"content", "text"}}},
	}
	
	_, err = currentEventsCollection.Indexes().CreateMany(ctx, eventIndexes)
	if err != nil {
		return fmt.Errorf("failed to create current events indexes: %w", err)
	}
	
	// Research sources indexes
	researchSourcesCollection := m.GetCollection("research_sources")
	sourceIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"type", 1}}},
		{Keys: bson.D{{"credibility", -1}}},
		{Keys: bson.D{{"relevance", -1}}},
		{Keys: bson.D{{"published_at", -1}}},
		{Keys: bson.D{{"keywords", 1}}},
		{Keys: bson.D{{"language", 1}}},
		{Keys: bson.D{{"url", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"title", "text"}, {"content", "text"}, {"summary", "text"}}},
	}
	
	_, err = researchSourcesCollection.Indexes().CreateMany(ctx, sourceIndexes)
	if err != nil {
		return fmt.Errorf("failed to create research sources indexes: %w", err)
	}
	
	return nil
}

// HealthCheck performs a health check on the MongoDB connection
func (m *MongoDB) HealthCheck(ctx context.Context) error {
	// Ping the database
	if err := m.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check if we can list collections (basic operation test)
	_, err := m.Database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("list collections failed: %w", err)
	}

	return nil
}
