package embedding

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmbeddingService defines the interface for embedding operations
type EmbeddingService interface {
	GenerateDocumentEmbedding(ctx context.Context, documentID primitive.ObjectID) error
	GenerateKnowledgeEmbedding(ctx context.Context, knowledgeID primitive.ObjectID) error
}

// EmbeddingRepository defines the interface for embedding repository operations
type EmbeddingRepository interface {
	GetDocumentsWithoutEmbeddings(ctx context.Context, limit int) ([]primitive.ObjectID, error)
	GetKnowledgeItemsWithoutEmbeddings(ctx context.Context, limit int) ([]primitive.ObjectID, error)
	GetEmbeddingStats(ctx context.Context) (*EmbeddingStats, error)
}
