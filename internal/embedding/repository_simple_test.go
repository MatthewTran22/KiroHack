package embedding

import (
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestNewRepositorySimple(t *testing.T) {
	// Test with nil database (should not panic)
	repo := NewRepository(nil)
	if repo == nil {
		t.Error("expected repository but got nil")
	}

	// Test with mock database
	var db *mongo.Database
	repo = NewRepository(db)
	if repo == nil {
		t.Error("expected repository but got nil")
	}
}

func TestEmbeddingStatsStruct(t *testing.T) {
	stats := &EmbeddingStats{
		DocumentsWithEmbeddings: 5,
		TotalDocuments:          10,
		KnowledgeWithEmbeddings: 3,
		TotalKnowledgeItems:     8,
	}

	if stats.DocumentsWithEmbeddings != 5 {
		t.Errorf("expected 5 documents with embeddings, got %d", stats.DocumentsWithEmbeddings)
	}
	if stats.TotalDocuments != 10 {
		t.Errorf("expected 10 total documents, got %d", stats.TotalDocuments)
	}
	if stats.KnowledgeWithEmbeddings != 3 {
		t.Errorf("expected 3 knowledge items with embeddings, got %d", stats.KnowledgeWithEmbeddings)
	}
	if stats.TotalKnowledgeItems != 8 {
		t.Errorf("expected 8 total knowledge items, got %d", stats.TotalKnowledgeItems)
	}
}
