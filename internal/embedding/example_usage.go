package embedding

import (
	"context"
	"fmt"
	"log"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExampleUsage demonstrates how to use the embedding service
func ExampleUsage() {
	// This is an example function showing how to use the embedding service
	// It's not meant to be run in production, just for documentation

	ctx := context.Background()

	// 1. Setup MongoDB connection
	mongoConfig := &database.Config{
		URI:          "mongodb://localhost:27017",
		DatabaseName: "ai_government_consultant",
	}

	mongodb, err := database.NewMongoDB(mongoConfig)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(ctx)

	// 2. Setup Redis connection (optional, for caching)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer redisClient.Close()

	// 3. Create logger
	testLogger := logger.NewTestLogger()

	// 4. Create embedding service
	embeddingConfig := &Config{
		GeminiAPIKey: "your-gemini-api-key-here", // Replace with actual API key
		MongoDB:      mongodb.Database,
		Redis:        redisClient,
		Logger:       testLogger,
	}

	service, err := NewService(embeddingConfig)
	if err != nil {
		log.Fatalf("Failed to create embedding service: %v", err)
	}

	// 5. Create repository and pipeline
	repo := NewRepository(mongodb.Database)
	pipelineConfig := &PipelineConfig{
		BatchSize:     10,
		MaxWorkers:    3,
		RetryAttempts: 2,
		RetryDelay:    5 * time.Second,
	}
	pipeline := NewPipeline(service, repo, testLogger, pipelineConfig)

	// Example 1: Generate embedding for text
	fmt.Println("=== Example 1: Generate Embedding ===")
	text := "This is a government policy document about digital transformation."
	embeddings, err := service.GenerateEmbedding(ctx, text)
	if err != nil {
		log.Printf("Failed to generate embedding: %v", err)
	} else {
		fmt.Printf("Generated embedding with %d dimensions\n", len(embeddings))
		fmt.Printf("First 5 values: %v\n", embeddings[:5])
	}

	// Example 2: Generate embedding for a document
	fmt.Println("\n=== Example 2: Generate Document Embedding ===")
	documentID := primitive.NewObjectID() // This would be a real document ID
	err = service.GenerateDocumentEmbedding(ctx, documentID)
	if err != nil {
		log.Printf("Failed to generate document embedding: %v", err)
	} else {
		fmt.Printf("Successfully generated embedding for document %s\n", documentID.Hex())
	}

	// Example 3: Perform vector search
	fmt.Println("\n=== Example 3: Vector Search ===")
	searchOptions := &SearchOptions{
		Limit:     5,
		Threshold: 0.7,
		Filters: map[string]interface{}{
			"metadata.category": "policy",
		},
	}

	results, err := service.VectorSearch(ctx, "digital transformation policy", searchOptions)
	if err != nil {
		log.Printf("Failed to perform vector search: %v", err)
	} else {
		fmt.Printf("Found %d search results\n", len(results))
		for i, result := range results {
			fmt.Printf("Result %d: ID=%s, Score=%.3f\n", i+1, result.ID, result.Score)
		}
	}

	// Example 4: Batch generate embeddings
	fmt.Println("\n=== Example 4: Batch Generate Embeddings ===")
	texts := []string{
		"Government policy on data protection",
		"Strategic planning for digital infrastructure",
		"Operational guidelines for remote work",
	}

	batchEmbeddings, err := service.BatchGenerateEmbeddings(ctx, texts)
	if err != nil {
		log.Printf("Failed to generate batch embeddings: %v", err)
	} else {
		fmt.Printf("Generated embeddings for %d texts\n", len(batchEmbeddings))
		for i, embedding := range batchEmbeddings {
			fmt.Printf("Text %d: %d dimensions\n", i+1, len(embedding))
		}
	}

	// Example 5: Process all documents without embeddings
	fmt.Println("\n=== Example 5: Process All Documents ===")
	result, err := pipeline.ProcessAllDocuments(ctx)
	if err != nil {
		log.Printf("Failed to process documents: %v", err)
	} else {
		fmt.Printf("Processing completed:\n")
		fmt.Printf("  Total processed: %d\n", result.TotalProcessed)
		fmt.Printf("  Successful: %d\n", result.Successful)
		fmt.Printf("  Failed: %d\n", result.Failed)
		fmt.Printf("  Duration: %v\n", result.Duration)
	}

	// Example 6: Get embedding statistics
	fmt.Println("\n=== Example 6: Embedding Statistics ===")
	stats, err := repo.GetEmbeddingStats(ctx)
	if err != nil {
		log.Printf("Failed to get embedding stats: %v", err)
	} else {
		fmt.Printf("Embedding Statistics:\n")
		fmt.Printf("  Documents with embeddings: %d/%d\n", stats.DocumentsWithEmbeddings, stats.TotalDocuments)
		fmt.Printf("  Knowledge items with embeddings: %d/%d\n", stats.KnowledgeWithEmbeddings, stats.TotalKnowledgeItems)
	}

	// Example 7: Find similar documents
	fmt.Println("\n=== Example 7: Find Similar Documents ===")
	sourceDocumentID := primitive.NewObjectID() // This would be a real document ID
	similarResults, err := service.GetSimilarDocuments(ctx, sourceDocumentID, 3)
	if err != nil {
		log.Printf("Failed to find similar documents: %v", err)
	} else {
		fmt.Printf("Found %d similar documents\n", len(similarResults))
		for i, result := range similarResults {
			fmt.Printf("Similar document %d: ID=%s, Score=%.3f\n", i+1, result.ID, result.Score)
		}
	}

	// Example 8: Clear embedding cache
	fmt.Println("\n=== Example 8: Clear Cache ===")
	err = service.ClearCache(ctx)
	if err != nil {
		log.Printf("Failed to clear cache: %v", err)
	} else {
		fmt.Println("Successfully cleared embedding cache")
	}

	fmt.Println("\n=== Example Usage Complete ===")
}

// ExampleConfiguration shows different ways to configure the embedding service
func ExampleConfiguration() {
	// Basic configuration
	basicConfig := &Config{
		GeminiAPIKey: "your-api-key",
		Logger:       logger.NewTestLogger(),
	}

	// Configuration with custom Gemini URL
	customURLConfig := &Config{
		GeminiAPIKey: "your-api-key",
		GeminiURL:    "https://custom-gemini-endpoint.com/v1/embeddings",
		Logger:       logger.NewTestLogger(),
	}

	// Configuration with MongoDB and Redis
	fullConfig := &Config{
		GeminiAPIKey: "your-api-key",
		MongoDB:      nil, // Would be actual MongoDB database
		Redis:        nil, // Would be actual Redis client
		Logger:       logger.NewTestLogger(),
	}

	// Pipeline configuration options
	fastPipelineConfig := &PipelineConfig{
		BatchSize:     100, // Larger batches
		MaxWorkers:    10,  // More workers
		RetryAttempts: 1,   // Fewer retries
		RetryDelay:    time.Second,
	}

	conservativePipelineConfig := &PipelineConfig{
		BatchSize:     10,               // Smaller batches
		MaxWorkers:    2,                // Fewer workers
		RetryAttempts: 5,                // More retries
		RetryDelay:    10 * time.Second, // Longer delays
	}

	// Use default configuration
	defaultPipelineConfig := DefaultPipelineConfig()

	// These configurations would be used like:
	_ = basicConfig
	_ = customURLConfig
	_ = fullConfig
	_ = fastPipelineConfig
	_ = conservativePipelineConfig
	_ = defaultPipelineConfig

	fmt.Println("Configuration examples created")
}

// ExampleErrorHandling shows how to handle common errors
func ExampleErrorHandling() {
	ctx := context.Background()

	// Example of handling service creation errors
	invalidConfig := &Config{
		// Missing GeminiAPIKey
		Logger: logger.NewTestLogger(),
	}

	service, err := NewService(invalidConfig)
	if err != nil {
		fmt.Printf("Service creation failed as expected: %v\n", err)
	}

	// Example of handling API errors
	if service != nil {
		_, err = service.GenerateEmbedding(ctx, "")
		if err != nil {
			fmt.Printf("Empty text embedding failed as expected: %v\n", err)
		}
	}

	// Example of handling search errors
	searchOptions := &SearchOptions{
		Limit:     -1,  // Invalid limit
		Threshold: 2.0, // Invalid threshold (should be 0-1)
	}

	if service != nil {
		_, err = service.VectorSearch(ctx, "test query", searchOptions)
		if err != nil {
			fmt.Printf("Invalid search options failed as expected: %v\n", err)
		}
	}

	fmt.Println("Error handling examples complete")
}
