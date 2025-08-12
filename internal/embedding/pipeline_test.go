package embedding

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-government-consultant/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mock service for testing
type mockEmbeddingService struct {
	generateDocumentEmbeddingFunc  func(ctx context.Context, documentID primitive.ObjectID) error
	generateKnowledgeEmbeddingFunc func(ctx context.Context, knowledgeID primitive.ObjectID) error
}

func (m *mockEmbeddingService) GenerateDocumentEmbedding(ctx context.Context, documentID primitive.ObjectID) error {
	if m.generateDocumentEmbeddingFunc != nil {
		return m.generateDocumentEmbeddingFunc(ctx, documentID)
	}
	return nil
}

func (m *mockEmbeddingService) GenerateKnowledgeEmbedding(ctx context.Context, knowledgeID primitive.ObjectID) error {
	if m.generateKnowledgeEmbeddingFunc != nil {
		return m.generateKnowledgeEmbeddingFunc(ctx, knowledgeID)
	}
	return nil
}

// Mock repository for testing
type mockEmbeddingRepository struct {
	getDocumentsWithoutEmbeddingsFunc      func(ctx context.Context, limit int) ([]primitive.ObjectID, error)
	getKnowledgeItemsWithoutEmbeddingsFunc func(ctx context.Context, limit int) ([]primitive.ObjectID, error)
	getEmbeddingStatsFunc                  func(ctx context.Context) (*EmbeddingStats, error)
}

func (m *mockEmbeddingRepository) GetDocumentsWithoutEmbeddings(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
	if m.getDocumentsWithoutEmbeddingsFunc != nil {
		return m.getDocumentsWithoutEmbeddingsFunc(ctx, limit)
	}
	return []primitive.ObjectID{}, nil
}

func (m *mockEmbeddingRepository) GetKnowledgeItemsWithoutEmbeddings(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
	if m.getKnowledgeItemsWithoutEmbeddingsFunc != nil {
		return m.getKnowledgeItemsWithoutEmbeddingsFunc(ctx, limit)
	}
	return []primitive.ObjectID{}, nil
}

func (m *mockEmbeddingRepository) GetEmbeddingStats(ctx context.Context) (*EmbeddingStats, error) {
	if m.getEmbeddingStatsFunc != nil {
		return m.getEmbeddingStatsFunc(ctx)
	}
	return &EmbeddingStats{}, nil
}

func TestNewPipeline(t *testing.T) {
	service := &mockEmbeddingService{}
	repo := &mockEmbeddingRepository{}
	logger := logger.NewTestLogger()

	tests := []struct {
		name   string
		config *PipelineConfig
	}{
		{
			name:   "with config",
			config: &PipelineConfig{BatchSize: 10, MaxWorkers: 2},
		},
		{
			name:   "with nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(service, repo, logger, tt.config)
			if pipeline == nil {
				t.Error("expected pipeline but got nil")
			}

			if tt.config == nil {
				// Should use default config
				if pipeline.batchSize != 50 {
					t.Errorf("expected default batch size 50, got %d", pipeline.batchSize)
				}
				if pipeline.maxWorkers != 5 {
					t.Errorf("expected default max workers 5, got %d", pipeline.maxWorkers)
				}
			} else {
				if pipeline.batchSize != tt.config.BatchSize {
					t.Errorf("expected batch size %d, got %d", tt.config.BatchSize, pipeline.batchSize)
				}
				if pipeline.maxWorkers != tt.config.MaxWorkers {
					t.Errorf("expected max workers %d, got %d", tt.config.MaxWorkers, pipeline.maxWorkers)
				}
			}
		})
	}
}

func TestProcessAllDocuments(t *testing.T) {
	tests := []struct {
		name                       string
		documentsWithoutEmbeddings [][]primitive.ObjectID
		generateEmbeddingError     error
		expectedSuccessful         int
		expectedFailed             int
		expectError                bool
	}{
		{
			name: "successful processing",
			documentsWithoutEmbeddings: [][]primitive.ObjectID{
				{primitive.NewObjectID(), primitive.NewObjectID()},
				{}, // Empty batch to signal completion
			},
			generateEmbeddingError: nil,
			expectedSuccessful:     2,
			expectedFailed:         0,
			expectError:            false,
		},
		{
			name: "partial failure",
			documentsWithoutEmbeddings: [][]primitive.ObjectID{
				{primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()},
				{}, // Empty batch to signal completion
			},
			generateEmbeddingError: errors.New("embedding generation failed"),
			expectedSuccessful:     0,
			expectedFailed:         3,
			expectError:            false,
		},
		{
			name: "no documents to process",
			documentsWithoutEmbeddings: [][]primitive.ObjectID{
				{}, // Empty batch immediately
			},
			generateEmbeddingError: nil,
			expectedSuccessful:     0,
			expectedFailed:         0,
			expectError:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			service := &mockEmbeddingService{
				generateDocumentEmbeddingFunc: func(ctx context.Context, documentID primitive.ObjectID) error {
					return tt.generateEmbeddingError
				},
			}

			repo := &mockEmbeddingRepository{
				getDocumentsWithoutEmbeddingsFunc: func(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
					if callCount >= len(tt.documentsWithoutEmbeddings) {
						return []primitive.ObjectID{}, nil
					}
					result := tt.documentsWithoutEmbeddings[callCount]
					callCount++
					return result, nil
				},
			}

			config := &PipelineConfig{
				BatchSize:     10,
				MaxWorkers:    2,
				RetryAttempts: 0, // No retries for faster tests
				RetryDelay:    time.Millisecond,
			}

			pipeline := NewPipeline(service, repo, logger.NewTestLogger(), config)

			ctx := context.Background()
			result, err := pipeline.ProcessAllDocuments(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Successful != tt.expectedSuccessful {
				t.Errorf("expected %d successful, got %d", tt.expectedSuccessful, result.Successful)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.TotalProcessed != tt.expectedSuccessful+tt.expectedFailed {
				t.Errorf("expected %d total processed, got %d",
					tt.expectedSuccessful+tt.expectedFailed, result.TotalProcessed)
			}
		})
	}
}

func TestProcessAllKnowledgeItems(t *testing.T) {
	tests := []struct {
		name                       string
		knowledgeWithoutEmbeddings [][]primitive.ObjectID
		generateEmbeddingError     error
		expectedSuccessful         int
		expectedFailed             int
		expectError                bool
	}{
		{
			name: "successful processing",
			knowledgeWithoutEmbeddings: [][]primitive.ObjectID{
				{primitive.NewObjectID(), primitive.NewObjectID()},
				{}, // Empty batch to signal completion
			},
			generateEmbeddingError: nil,
			expectedSuccessful:     2,
			expectedFailed:         0,
			expectError:            false,
		},
		{
			name: "partial failure",
			knowledgeWithoutEmbeddings: [][]primitive.ObjectID{
				{primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()},
				{}, // Empty batch to signal completion
			},
			generateEmbeddingError: errors.New("embedding generation failed"),
			expectedSuccessful:     0,
			expectedFailed:         3,
			expectError:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			service := &mockEmbeddingService{
				generateKnowledgeEmbeddingFunc: func(ctx context.Context, knowledgeID primitive.ObjectID) error {
					return tt.generateEmbeddingError
				},
			}

			repo := &mockEmbeddingRepository{
				getKnowledgeItemsWithoutEmbeddingsFunc: func(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
					if callCount >= len(tt.knowledgeWithoutEmbeddings) {
						return []primitive.ObjectID{}, nil
					}
					result := tt.knowledgeWithoutEmbeddings[callCount]
					callCount++
					return result, nil
				},
			}

			config := &PipelineConfig{
				BatchSize:     10,
				MaxWorkers:    2,
				RetryAttempts: 0, // No retries for faster tests
				RetryDelay:    time.Millisecond,
			}

			pipeline := NewPipeline(service, repo, logger.NewTestLogger(), config)

			ctx := context.Background()
			result, err := pipeline.ProcessAllKnowledgeItems(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Successful != tt.expectedSuccessful {
				t.Errorf("expected %d successful, got %d", tt.expectedSuccessful, result.Successful)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
		})
	}
}

func TestProcessSpecificDocuments(t *testing.T) {
	documentIDs := []primitive.ObjectID{
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
	}

	tests := []struct {
		name                   string
		generateEmbeddingError error
		expectedSuccessful     int
		expectedFailed         int
	}{
		{
			name:                   "all successful",
			generateEmbeddingError: nil,
			expectedSuccessful:     3,
			expectedFailed:         0,
		},
		{
			name:                   "all failed",
			generateEmbeddingError: errors.New("embedding generation failed"),
			expectedSuccessful:     0,
			expectedFailed:         3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockEmbeddingService{
				generateDocumentEmbeddingFunc: func(ctx context.Context, documentID primitive.ObjectID) error {
					return tt.generateEmbeddingError
				},
			}

			repo := &mockEmbeddingRepository{}

			config := &PipelineConfig{
				BatchSize:     10,
				MaxWorkers:    2,
				RetryAttempts: 0, // No retries for faster tests
				RetryDelay:    time.Millisecond,
			}

			pipeline := NewPipeline(service, repo, logger.NewTestLogger(), config)

			ctx := context.Background()
			result, err := pipeline.ProcessSpecificDocuments(ctx, documentIDs)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Successful != tt.expectedSuccessful {
				t.Errorf("expected %d successful, got %d", tt.expectedSuccessful, result.Successful)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.TotalProcessed != len(documentIDs) {
				t.Errorf("expected %d total processed, got %d", len(documentIDs), result.TotalProcessed)
			}
		})
	}
}

func TestProcessSpecificKnowledgeItems(t *testing.T) {
	knowledgeIDs := []primitive.ObjectID{
		primitive.NewObjectID(),
		primitive.NewObjectID(),
	}

	tests := []struct {
		name                   string
		generateEmbeddingError error
		expectedSuccessful     int
		expectedFailed         int
	}{
		{
			name:                   "all successful",
			generateEmbeddingError: nil,
			expectedSuccessful:     2,
			expectedFailed:         0,
		},
		{
			name:                   "all failed",
			generateEmbeddingError: errors.New("embedding generation failed"),
			expectedSuccessful:     0,
			expectedFailed:         2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockEmbeddingService{
				generateKnowledgeEmbeddingFunc: func(ctx context.Context, knowledgeID primitive.ObjectID) error {
					return tt.generateEmbeddingError
				},
			}

			repo := &mockEmbeddingRepository{}

			config := &PipelineConfig{
				BatchSize:     10,
				MaxWorkers:    2,
				RetryAttempts: 0, // No retries for faster tests
				RetryDelay:    time.Millisecond,
			}

			pipeline := NewPipeline(service, repo, logger.NewTestLogger(), config)

			ctx := context.Background()
			result, err := pipeline.ProcessSpecificKnowledgeItems(ctx, knowledgeIDs)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Successful != tt.expectedSuccessful {
				t.Errorf("expected %d successful, got %d", tt.expectedSuccessful, result.Successful)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.TotalProcessed != len(knowledgeIDs) {
				t.Errorf("expected %d total processed, got %d", len(knowledgeIDs), result.TotalProcessed)
			}
		})
	}
}

func TestGetProcessingStats(t *testing.T) {
	expectedStats := &EmbeddingStats{
		DocumentsWithEmbeddings: 5,
		TotalDocuments:          10,
		KnowledgeWithEmbeddings: 3,
		TotalKnowledgeItems:     8,
	}

	service := &mockEmbeddingService{}
	repo := &mockEmbeddingRepository{
		getEmbeddingStatsFunc: func(ctx context.Context) (*EmbeddingStats, error) {
			return expectedStats, nil
		},
	}

	pipeline := NewPipeline(service, repo, logger.NewTestLogger(), nil)

	ctx := context.Background()
	stats, err := pipeline.GetProcessingStats(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if stats.DocumentsWithEmbeddings != expectedStats.DocumentsWithEmbeddings {
		t.Errorf("expected %d documents with embeddings, got %d",
			expectedStats.DocumentsWithEmbeddings, stats.DocumentsWithEmbeddings)
	}
	if stats.TotalDocuments != expectedStats.TotalDocuments {
		t.Errorf("expected %d total documents, got %d",
			expectedStats.TotalDocuments, stats.TotalDocuments)
	}
	if stats.KnowledgeWithEmbeddings != expectedStats.KnowledgeWithEmbeddings {
		t.Errorf("expected %d knowledge items with embeddings, got %d",
			expectedStats.KnowledgeWithEmbeddings, stats.KnowledgeWithEmbeddings)
	}
	if stats.TotalKnowledgeItems != expectedStats.TotalKnowledgeItems {
		t.Errorf("expected %d total knowledge items, got %d",
			expectedStats.TotalKnowledgeItems, stats.TotalKnowledgeItems)
	}
}

func TestProcessWithRetry(t *testing.T) {
	tests := []struct {
		name          string
		failureCount  int
		retryAttempts int
		expectSuccess bool
	}{
		{
			name:          "success on first try",
			failureCount:  0,
			retryAttempts: 3,
			expectSuccess: true,
		},
		{
			name:          "success on retry",
			failureCount:  2,
			retryAttempts: 3,
			expectSuccess: true,
		},
		{
			name:          "failure after all retries",
			failureCount:  5,
			retryAttempts: 3,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			service := &mockEmbeddingService{
				generateDocumentEmbeddingFunc: func(ctx context.Context, documentID primitive.ObjectID) error {
					callCount++
					if callCount <= tt.failureCount {
						return errors.New("temporary failure")
					}
					return nil
				},
			}

			repo := &mockEmbeddingRepository{}

			config := &PipelineConfig{
				BatchSize:     10,
				MaxWorkers:    1,
				RetryAttempts: tt.retryAttempts,
				RetryDelay:    time.Millisecond, // Fast retry for tests
			}

			pipeline := NewPipeline(service, repo, logger.NewTestLogger(), config)

			ctx := context.Background()
			documentID := primitive.NewObjectID()
			err := pipeline.processDocumentWithRetry(ctx, documentID)

			if tt.expectSuccess && err != nil {
				t.Errorf("expected success but got error: %v", err)
			}
			if !tt.expectSuccess && err == nil {
				t.Error("expected failure but got success")
			}

			expectedCalls := tt.failureCount + 1
			if !tt.expectSuccess {
				expectedCalls = tt.retryAttempts + 1
			}
			if callCount != expectedCalls {
				t.Errorf("expected %d calls, got %d", expectedCalls, callCount)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	service := &mockEmbeddingService{
		generateDocumentEmbeddingFunc: func(ctx context.Context, documentID primitive.ObjectID) error {
			// Simulate slow operation
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	repo := &mockEmbeddingRepository{
		getDocumentsWithoutEmbeddingsFunc: func(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
			return []primitive.ObjectID{primitive.NewObjectID()}, nil
		},
	}

	config := &PipelineConfig{
		BatchSize:     1,
		MaxWorkers:    1,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	}

	pipeline := NewPipeline(service, repo, logger.NewTestLogger(), config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := pipeline.ProcessAllDocuments(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected context deadline exceeded, got: %v", err)
	}

	// Should have partial results
	if result == nil {
		t.Error("expected result even with cancellation")
	}
}
