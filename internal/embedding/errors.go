package embedding

import "errors"

// Embedding service errors
var (
	// API errors
	ErrAPIKeyRequired     = errors.New("gemini API key is required")
	ErrAPIRequestFailed   = errors.New("API request failed")
	ErrAPIResponseInvalid = errors.New("API response is invalid")
	ErrAPIRateLimited     = errors.New("API rate limit exceeded")

	// Embedding errors
	ErrEmbeddingGeneration = errors.New("failed to generate embedding")
	ErrEmbeddingNotFound   = errors.New("embedding not found")
	ErrEmbeddingInvalid    = errors.New("embedding is invalid")
	ErrEmbeddingDimension  = errors.New("embedding dimension mismatch")

	// Search errors
	ErrSearchQueryEmpty    = errors.New("search query is empty")
	ErrSearchNoResults     = errors.New("no search results found")
	ErrSearchInvalidFilter = errors.New("invalid search filter")
	ErrSearchThreshold     = errors.New("invalid similarity threshold")

	// Database errors
	ErrDocumentNotFound    = errors.New("document not found")
	ErrKnowledgeNotFound   = errors.New("knowledge item not found")
	ErrDatabaseConnection  = errors.New("database connection failed")
	ErrIndexCreationFailed = errors.New("vector index creation failed")

	// Cache errors
	ErrCacheConnection = errors.New("cache connection failed")
	ErrCacheOperation  = errors.New("cache operation failed")

	// Pipeline errors
	ErrPipelineProcessing = errors.New("pipeline processing failed")
	ErrWorkerPoolFailed   = errors.New("worker pool failed")
	ErrBatchProcessing    = errors.New("batch processing failed")
)
