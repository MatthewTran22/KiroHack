# Embedding Service

The embedding service provides vector embedding generation and semantic search capabilities for the AI Government Consultant platform. It integrates with Google's Gemini API for text vectorization and MongoDB for vector storage and search.

## Features

- **Text Embedding Generation**: Convert text to high-dimensional vectors using Gemini 2.5 Flash
- **Document Embedding**: Generate embeddings for uploaded documents
- **Knowledge Item Embedding**: Generate embeddings for knowledge base items
- **Vector Search**: Semantic similarity search across documents and knowledge items
- **Batch Processing**: Efficient batch processing with worker pools and retry logic
- **Caching**: Redis-based caching for improved performance
- **MongoDB Integration**: Vector storage and indexing using MongoDB
- **Pipeline Processing**: Automated processing of items without embeddings

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Service       │    │   Repository    │    │   Pipeline      │
│                 │    │                 │    │                 │
│ - Generate      │    │ - CRUD Ops      │    │ - Batch Proc    │
│ - Search        │    │ - Stats         │    │ - Worker Pool   │
│ - Cache         │    │ - Indexes       │    │ - Retry Logic   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │   Gemini API    │    │   MongoDB       │    │   Redis Cache   │
         │                 │    │                 │    │                 │
         │ - Text to Vec   │    │ - Vector Store  │    │ - Embed Cache   │
         │ - Embeddings    │    │ - Search Index  │    │ - Performance   │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Components

### Service (`service.go`)
- Main embedding service with Gemini API integration
- Text-to-vector conversion
- Document and knowledge item embedding generation
- Vector similarity search
- Batch processing capabilities
- Redis caching integration

### Repository (`repository.go`)
- Database operations for embeddings
- MongoDB vector index management
- Statistics and analytics
- CRUD operations for embedding data

### Pipeline (`pipeline.go`)
- Batch processing with worker pools
- Retry logic for failed operations
- Progress tracking and reporting
- Concurrent processing of multiple items

## Usage

### Basic Setup

```go
import (
    "ai-government-consultant/internal/embedding"
    "ai-government-consultant/internal/database"
    "ai-government-consultant/pkg/logger"
)

// Setup MongoDB
mongoConfig := &database.Config{
    URI:          "mongodb://localhost:27017",
    DatabaseName: "ai_government_consultant",
}
mongodb, err := database.NewMongoDB(mongoConfig)
if err != nil {
    log.Fatal(err)
}

// Setup Redis (optional)
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create embedding service
config := &embedding.Config{
    GeminiAPIKey: "your-gemini-api-key",
    MongoDB:      mongodb.Database,
    Redis:        redisClient,
    Logger:       logger.New(logConfig),
}

service, err := embedding.NewService(config)
if err != nil {
    log.Fatal(err)
}
```

### Generate Embeddings

```go
// Generate embedding for text
text := "Government policy on digital transformation"
embeddings, err := service.GenerateEmbedding(ctx, text)
if err != nil {
    log.Fatal(err)
}

// Generate embedding for document
documentID := primitive.NewObjectID()
err = service.GenerateDocumentEmbedding(ctx, documentID)
if err != nil {
    log.Fatal(err)
}

// Batch generate embeddings
texts := []string{
    "Policy document 1",
    "Policy document 2",
    "Policy document 3",
}
batchEmbeddings, err := service.BatchGenerateEmbeddings(ctx, texts)
if err != nil {
    log.Fatal(err)
}
```

### Vector Search

```go
// Basic search
results, err := service.VectorSearch(ctx, "digital transformation", nil)
if err != nil {
    log.Fatal(err)
}

// Advanced search with options
options := &embedding.SearchOptions{
    Limit:     10,
    Threshold: 0.7,
    Collection: "documents",
    Filters: map[string]interface{}{
        "metadata.category": "policy",
        "classification.level": "PUBLIC",
    },
}

results, err = service.VectorSearch(ctx, "digital transformation", options)
if err != nil {
    log.Fatal(err)
}

// Process results
for _, result := range results {
    fmt.Printf("ID: %s, Score: %.3f\n", result.ID, result.Score)
    if result.Document != nil {
        fmt.Printf("Document: %s\n", result.Document.Name)
    }
}
```

### Pipeline Processing

```go
// Create repository and pipeline
repo := embedding.NewRepository(mongodb.Database)
pipelineConfig := &embedding.PipelineConfig{
    BatchSize:     50,
    MaxWorkers:    5,
    RetryAttempts: 3,
    RetryDelay:    5 * time.Second,
}

pipeline := embedding.NewPipeline(service, repo, logger, pipelineConfig)

// Process all documents without embeddings
result, err := pipeline.ProcessAllDocuments(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Processed: %d, Successful: %d, Failed: %d\n", 
    result.TotalProcessed, result.Successful, result.Failed)

// Process specific documents
documentIDs := []primitive.ObjectID{id1, id2, id3}
result, err = pipeline.ProcessSpecificDocuments(ctx, documentIDs)
if err != nil {
    log.Fatal(err)
}
```

### Statistics and Monitoring

```go
// Get embedding statistics
stats, err := repo.GetEmbeddingStats(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Documents with embeddings: %d/%d\n", 
    stats.DocumentsWithEmbeddings, stats.TotalDocuments)
fmt.Printf("Knowledge items with embeddings: %d/%d\n", 
    stats.KnowledgeWithEmbeddings, stats.TotalKnowledgeItems)
```

## Configuration

### Service Configuration

```go
type Config struct {
    GeminiAPIKey string           // Required: Gemini API key
    GeminiURL    string           // Optional: Custom Gemini endpoint
    MongoDB      *mongo.Database  // Optional: MongoDB database
    Redis        *redis.Client    // Optional: Redis client for caching
    Logger       logger.Logger    // Required: Logger instance
}
```

### Pipeline Configuration

```go
type PipelineConfig struct {
    BatchSize     int           // Number of items per batch (default: 50)
    MaxWorkers    int           // Maximum concurrent workers (default: 5)
    RetryAttempts int           // Number of retry attempts (default: 3)
    RetryDelay    time.Duration // Delay between retries (default: 5s)
}
```

### Search Options

```go
type SearchOptions struct {
    Limit      int                    // Maximum results (default: 10)
    Threshold  float64                // Similarity threshold 0-1 (default: 0.7)
    Collection string                 // "documents" or "knowledge_items"
    Filters    map[string]interface{} // MongoDB filters
}
```

## Vector Search

The service supports semantic similarity search using cosine similarity. Search results are ranked by similarity score (0-1, where 1 is most similar).

### Search Collections
- **documents**: Search across uploaded documents
- **knowledge_items**: Search across knowledge base items
- **both**: Search across both collections (default)

### Filtering
You can filter search results using MongoDB query syntax:

```go
filters := map[string]interface{}{
    "metadata.category": "policy",
    "classification.level": bson.M{"$in": []string{"PUBLIC", "INTERNAL"}},
    "metadata.tags": bson.M{"$in": []string{"digital", "transformation"}},
    "uploaded_at": bson.M{"$gte": time.Now().AddDate(0, -6, 0)},
}
```

## Performance Optimization

### Caching
- Embeddings are cached in Redis with 24-hour TTL
- Cache keys are based on text content hash
- Automatic cache invalidation and cleanup

### Batch Processing
- Process multiple items concurrently
- Configurable worker pools
- Automatic retry with exponential backoff
- Progress tracking and error reporting

### Database Optimization
- Vector indexes for fast similarity search
- Compound indexes for filtered searches
- Efficient aggregation pipelines
- Connection pooling and timeout management

## Error Handling

The service provides comprehensive error handling:

```go
// API errors
ErrAPIKeyRequired     = errors.New("gemini API key is required")
ErrAPIRequestFailed   = errors.New("API request failed")
ErrAPIRateLimited     = errors.New("API rate limit exceeded")

// Embedding errors
ErrEmbeddingGeneration = errors.New("failed to generate embedding")
ErrEmbeddingNotFound   = errors.New("embedding not found")
ErrEmbeddingDimension  = errors.New("embedding dimension mismatch")

// Search errors
ErrSearchQueryEmpty    = errors.New("search query is empty")
ErrSearchNoResults     = errors.New("no search results found")
ErrSearchThreshold     = errors.New("invalid similarity threshold")
```

## Testing

### Unit Tests
```bash
go test ./internal/embedding/... -v
```

### Integration Tests
```bash
# Set environment variables
export LLM_API_KEY="your-gemini-api-key"
export MONGO_URI="mongodb://localhost:27017"
export REDIS_HOST="localhost:6379"

# Test embedding functionality through API endpoints
```

### Docker Testing
The integration tests are designed to work with Docker containers:

```bash
# Start MongoDB and Redis
docker-compose up -d mongodb redis

# Test embedding functionality through API endpoints

# Stop containers
docker-compose down
```

## MongoDB Vector Search

For production deployments, consider using MongoDB Atlas with vector search capabilities:

1. **Atlas Vector Search**: Native vector search with optimized indexes
2. **Search Indexes**: Create vector search indexes through Atlas UI
3. **Performance**: Better performance than aggregation-based similarity search
4. **Scalability**: Handles large-scale vector operations efficiently

## Security Considerations

- **API Keys**: Store Gemini API keys securely (environment variables, secrets management)
- **Data Classification**: Respect document security classifications in search results
- **Access Control**: Implement proper authorization for embedding operations
- **Audit Logging**: Log all embedding generation and search operations
- **Rate Limiting**: Implement rate limiting for API calls to prevent abuse

## Monitoring and Observability

- **Metrics**: Track embedding generation rates, search performance, cache hit rates
- **Logging**: Comprehensive logging with structured fields
- **Health Checks**: Monitor service health and external dependencies
- **Alerting**: Set up alerts for API failures, high error rates, performance degradation

## Troubleshooting

### Common Issues

1. **API Key Issues**
   - Verify Gemini API key is valid and has sufficient quota
   - Check API endpoint URL and network connectivity

2. **MongoDB Connection**
   - Verify MongoDB connection string and credentials
   - Check network connectivity and firewall rules

3. **Redis Connection**
   - Verify Redis connection parameters
   - Check if Redis is running and accessible

4. **Performance Issues**
   - Monitor API rate limits and quotas
   - Check MongoDB index usage and query performance
   - Verify Redis cache hit rates

5. **Vector Search Issues**
   - Ensure documents have embeddings before searching
   - Check similarity thresholds (too high = no results)
   - Verify search filters are valid MongoDB queries

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
// Set log level to debug
config := config.LoggingConfig{
    Level:  "debug",
    Format: "json",
    Output: "stdout",
}
logger := logger.New(config)
```

## Future Enhancements

- **Multiple Embedding Models**: Support for different embedding models
- **Hybrid Search**: Combine vector search with traditional text search
- **Real-time Updates**: Real-time embedding updates for modified documents
- **Advanced Analytics**: Embedding quality metrics and analytics
- **Multi-language Support**: Language-specific embedding models
- **Federated Search**: Search across multiple embedding spaces