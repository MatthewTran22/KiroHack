package embedding

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ai-government-consultant/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Pipeline handles batch processing of embeddings
type Pipeline struct {
	service    EmbeddingService
	repository EmbeddingRepository
	logger     logger.Logger

	// Configuration
	batchSize     int
	maxWorkers    int
	retryAttempts int
	retryDelay    time.Duration
}

// PipelineConfig holds configuration for the embedding pipeline
type PipelineConfig struct {
	BatchSize     int
	MaxWorkers    int
	RetryAttempts int
	RetryDelay    time.Duration
}

// DefaultPipelineConfig returns default pipeline configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		BatchSize:     50,
		MaxWorkers:    5,
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
	}
}

// NewPipeline creates a new embedding pipeline
func NewPipeline(service EmbeddingService, repository EmbeddingRepository, logger logger.Logger, config *PipelineConfig) *Pipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}

	return &Pipeline{
		service:       service,
		repository:    repository,
		logger:        logger,
		batchSize:     config.BatchSize,
		maxWorkers:    config.MaxWorkers,
		retryAttempts: config.RetryAttempts,
		retryDelay:    config.RetryDelay,
	}
}

// ProcessResult represents the result of processing embeddings
type ProcessResult struct {
	TotalProcessed int           `json:"total_processed"`
	Successful     int           `json:"successful"`
	Failed         int           `json:"failed"`
	Errors         []string      `json:"errors"`
	Duration       time.Duration `json:"duration"`
}

// ProcessAllDocuments processes embeddings for all documents that don't have them
func (p *Pipeline) ProcessAllDocuments(ctx context.Context) (*ProcessResult, error) {
	startTime := time.Now()
	result := &ProcessResult{}

	p.logger.Info("Starting document embedding processing pipeline", nil)

	for {
		// Get batch of documents without embeddings
		documentIDs, err := p.repository.GetDocumentsWithoutEmbeddings(ctx, p.batchSize)
		if err != nil {
			return result, fmt.Errorf("failed to get documents without embeddings: %w", err)
		}

		if len(documentIDs) == 0 {
			break // No more documents to process
		}

		p.logger.Info("Processing document batch", map[string]interface{}{
			"batch_size": len(documentIDs),
		})

		// Process batch with workers
		batchResult := p.processDocumentBatch(ctx, documentIDs)
		result.TotalProcessed += batchResult.TotalProcessed
		result.Successful += batchResult.Successful
		result.Failed += batchResult.Failed
		result.Errors = append(result.Errors, batchResult.Errors...)

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}
	}

	result.Duration = time.Since(startTime)
	p.logger.Info("Document embedding processing completed", map[string]interface{}{
		"total_processed": result.TotalProcessed,
		"successful":      result.Successful,
		"failed":          result.Failed,
		"duration":        result.Duration.String(),
	})

	return result, nil
}

// ProcessAllKnowledgeItems processes embeddings for all knowledge items that don't have them
func (p *Pipeline) ProcessAllKnowledgeItems(ctx context.Context) (*ProcessResult, error) {
	startTime := time.Now()
	result := &ProcessResult{}

	p.logger.Info("Starting knowledge item embedding processing pipeline", nil)

	for {
		// Get batch of knowledge items without embeddings
		knowledgeIDs, err := p.repository.GetKnowledgeItemsWithoutEmbeddings(ctx, p.batchSize)
		if err != nil {
			return result, fmt.Errorf("failed to get knowledge items without embeddings: %w", err)
		}

		if len(knowledgeIDs) == 0 {
			break // No more knowledge items to process
		}

		p.logger.Info("Processing knowledge batch", map[string]interface{}{
			"batch_size": len(knowledgeIDs),
		})

		// Process batch with workers
		batchResult := p.processKnowledgeBatch(ctx, knowledgeIDs)
		result.TotalProcessed += batchResult.TotalProcessed
		result.Successful += batchResult.Successful
		result.Failed += batchResult.Failed
		result.Errors = append(result.Errors, batchResult.Errors...)

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}
	}

	result.Duration = time.Since(startTime)
	p.logger.Info("Knowledge item embedding processing completed", map[string]interface{}{
		"total_processed": result.TotalProcessed,
		"successful":      result.Successful,
		"failed":          result.Failed,
		"duration":        result.Duration.String(),
	})

	return result, nil
}

// processDocumentBatch processes a batch of documents with worker pool
func (p *Pipeline) processDocumentBatch(ctx context.Context, documentIDs []primitive.ObjectID) *ProcessResult {
	result := &ProcessResult{TotalProcessed: len(documentIDs)}

	// Create worker pool
	jobs := make(chan primitive.ObjectID, len(documentIDs))
	results := make(chan error, len(documentIDs))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < p.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for documentID := range jobs {
				err := p.processDocumentWithRetry(ctx, documentID)
				results <- err
			}
		}()
	}

	// Send jobs
	for _, documentID := range documentIDs {
		jobs <- documentID
	}
	close(jobs)

	// Wait for workers to complete
	wg.Wait()
	close(results)

	// Collect results
	for err := range results {
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Successful++
		}
	}

	return result
}

// processKnowledgeBatch processes a batch of knowledge items with worker pool
func (p *Pipeline) processKnowledgeBatch(ctx context.Context, knowledgeIDs []primitive.ObjectID) *ProcessResult {
	result := &ProcessResult{TotalProcessed: len(knowledgeIDs)}

	// Create worker pool
	jobs := make(chan primitive.ObjectID, len(knowledgeIDs))
	results := make(chan error, len(knowledgeIDs))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < p.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for knowledgeID := range jobs {
				err := p.processKnowledgeWithRetry(ctx, knowledgeID)
				results <- err
			}
		}()
	}

	// Send jobs
	for _, knowledgeID := range knowledgeIDs {
		jobs <- knowledgeID
	}
	close(jobs)

	// Wait for workers to complete
	wg.Wait()
	close(results)

	// Collect results
	for err := range results {
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Successful++
		}
	}

	return result
}

// processDocumentWithRetry processes a single document with retry logic
func (p *Pipeline) processDocumentWithRetry(ctx context.Context, documentID primitive.ObjectID) error {
	var lastErr error

	for attempt := 0; attempt <= p.retryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.retryDelay):
			}

			p.logger.Debug("Retrying document embedding generation", map[string]interface{}{
				"document_id": documentID.Hex(),
				"attempt":     attempt,
			})
		}

		err := p.service.GenerateDocumentEmbedding(ctx, documentID)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		p.logger.Error("Failed to generate document embedding", err, map[string]interface{}{
			"document_id": documentID.Hex(),
			"attempt":     attempt,
		})
	}

	return fmt.Errorf("failed to process document %s after %d attempts: %w",
		documentID.Hex(), p.retryAttempts+1, lastErr)
}

// processKnowledgeWithRetry processes a single knowledge item with retry logic
func (p *Pipeline) processKnowledgeWithRetry(ctx context.Context, knowledgeID primitive.ObjectID) error {
	var lastErr error

	for attempt := 0; attempt <= p.retryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.retryDelay):
			}

			p.logger.Debug("Retrying knowledge embedding generation", map[string]interface{}{
				"knowledge_id": knowledgeID.Hex(),
				"attempt":      attempt,
			})
		}

		err := p.service.GenerateKnowledgeEmbedding(ctx, knowledgeID)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		p.logger.Error("Failed to generate knowledge embedding", err, map[string]interface{}{
			"knowledge_id": knowledgeID.Hex(),
			"attempt":      attempt,
		})
	}

	return fmt.Errorf("failed to process knowledge item %s after %d attempts: %w",
		knowledgeID.Hex(), p.retryAttempts+1, lastErr)
}

// ProcessSpecificDocuments processes embeddings for specific documents
func (p *Pipeline) ProcessSpecificDocuments(ctx context.Context, documentIDs []primitive.ObjectID) (*ProcessResult, error) {
	startTime := time.Now()

	p.logger.Info("Processing specific documents", map[string]interface{}{
		"count": len(documentIDs),
	})

	result := p.processDocumentBatch(ctx, documentIDs)
	result.Duration = time.Since(startTime)

	p.logger.Info("Specific document processing completed", map[string]interface{}{
		"total_processed": result.TotalProcessed,
		"successful":      result.Successful,
		"failed":          result.Failed,
		"duration":        result.Duration.String(),
	})

	return result, nil
}

// ProcessSpecificKnowledgeItems processes embeddings for specific knowledge items
func (p *Pipeline) ProcessSpecificKnowledgeItems(ctx context.Context, knowledgeIDs []primitive.ObjectID) (*ProcessResult, error) {
	startTime := time.Now()

	p.logger.Info("Processing specific knowledge items", map[string]interface{}{
		"count": len(knowledgeIDs),
	})

	result := p.processKnowledgeBatch(ctx, knowledgeIDs)
	result.Duration = time.Since(startTime)

	p.logger.Info("Specific knowledge processing completed", map[string]interface{}{
		"total_processed": result.TotalProcessed,
		"successful":      result.Successful,
		"failed":          result.Failed,
		"duration":        result.Duration.String(),
	})

	return result, nil
}

// GetProcessingStats returns current processing statistics
func (p *Pipeline) GetProcessingStats(ctx context.Context) (*EmbeddingStats, error) {
	return p.repository.GetEmbeddingStats(ctx)
}
