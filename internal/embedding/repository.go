package embedding

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for embeddings
type Repository struct {
	mongodb *mongo.Database
}

// NewRepository creates a new embedding repository
func NewRepository(mongodb *mongo.Database) *Repository {
	return &Repository{
		mongodb: mongodb,
	}
}

// CreateVectorSearchIndexes creates vector search indexes for MongoDB Atlas
// Note: This requires MongoDB Atlas with vector search capabilities
func (r *Repository) CreateVectorSearchIndexes(ctx context.Context) error {
	// Create vector search index for documents
	if err := r.createDocumentVectorIndex(ctx); err != nil {
		return fmt.Errorf("failed to create document vector index: %w", err)
	}

	// Create vector search index for knowledge items
	if err := r.createKnowledgeVectorIndex(ctx); err != nil {
		return fmt.Errorf("failed to create knowledge vector index: %w", err)
	}

	return nil
}

// createDocumentVectorIndex creates a vector search index for documents collection
func (r *Repository) createDocumentVectorIndex(ctx context.Context) error {
	collection := r.mongodb.Collection("documents")

	// Check if index already exists
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var index bson.M
		if err := cursor.Decode(&index); err != nil {
			continue
		}
		if name, ok := index["name"].(string); ok && name == "document_vector_index" {
			return nil // Index already exists
		}
	}

	// Create vector search index
	// Note: In a real MongoDB Atlas environment, you would use the Atlas UI or Atlas CLI
	// to create vector search indexes. This is a placeholder for the index structure.
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "embeddings", Value: "vector"},
		},
		Options: options.Index().
			SetName("document_vector_index").
			SetBackground(true),
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		// Vector search indexes might not be supported in local MongoDB
		// Log the error but don't fail the operation
		return fmt.Errorf("vector search index creation failed (this is expected for local MongoDB): %w", err)
	}

	return nil
}

// createKnowledgeVectorIndex creates a vector search index for knowledge_items collection
func (r *Repository) createKnowledgeVectorIndex(ctx context.Context) error {
	collection := r.mongodb.Collection("knowledge_items")

	// Check if index already exists
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var index bson.M
		if err := cursor.Decode(&index); err != nil {
			continue
		}
		if name, ok := index["name"].(string); ok && name == "knowledge_vector_index" {
			return nil // Index already exists
		}
	}

	// Create vector search index
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "embeddings", Value: "vector"},
		},
		Options: options.Index().
			SetName("knowledge_vector_index").
			SetBackground(true),
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		// Vector search indexes might not be supported in local MongoDB
		// Log the error but don't fail the operation
		return fmt.Errorf("vector search index creation failed (this is expected for local MongoDB): %w", err)
	}

	return nil
}

// GetDocumentsWithoutEmbeddings retrieves documents that don't have embeddings
func (r *Repository) GetDocumentsWithoutEmbeddings(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
	collection := r.mongodb.Collection("documents")

	filter := bson.M{
		"processing_status": "completed",
		"$or": []bson.M{
			{"embeddings": bson.M{"$exists": false}},
			{"embeddings": bson.M{"$size": 0}},
			{"embeddings": nil},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetProjection(bson.M{"_id": 1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents without embeddings: %w", err)
	}
	defer cursor.Close(ctx)

	var documentIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var doc struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		documentIDs = append(documentIDs, doc.ID)
	}

	return documentIDs, nil
}

// GetKnowledgeItemsWithoutEmbeddings retrieves knowledge items that don't have embeddings
func (r *Repository) GetKnowledgeItemsWithoutEmbeddings(ctx context.Context, limit int) ([]primitive.ObjectID, error) {
	collection := r.mongodb.Collection("knowledge_items")

	filter := bson.M{
		"is_active": true,
		"$or": []bson.M{
			{"embeddings": bson.M{"$exists": false}},
			{"embeddings": bson.M{"$size": 0}},
			{"embeddings": nil},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetProjection(bson.M{"_id": 1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge items without embeddings: %w", err)
	}
	defer cursor.Close(ctx)

	var knowledgeIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var knowledge struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&knowledge); err != nil {
			continue
		}
		knowledgeIDs = append(knowledgeIDs, knowledge.ID)
	}

	return knowledgeIDs, nil
}

// UpdateDocumentEmbeddings updates a document with its embeddings
func (r *Repository) UpdateDocumentEmbeddings(ctx context.Context, documentID primitive.ObjectID, embeddings []float64) error {
	collection := r.mongodb.Collection("documents")

	update := bson.M{
		"$set": bson.M{
			"embeddings":           embeddings,
			"processing_timestamp": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
	if err != nil {
		return fmt.Errorf("failed to update document embeddings: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found: %s", documentID.Hex())
	}

	return nil
}

// UpdateKnowledgeEmbeddings updates a knowledge item with its embeddings
func (r *Repository) UpdateKnowledgeEmbeddings(ctx context.Context, knowledgeID primitive.ObjectID, embeddings []float64) error {
	collection := r.mongodb.Collection("knowledge_items")

	update := bson.M{
		"$set": bson.M{
			"embeddings": embeddings,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": knowledgeID}, update)
	if err != nil {
		return fmt.Errorf("failed to update knowledge embeddings: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("knowledge item not found: %s", knowledgeID.Hex())
	}

	return nil
}

// GetEmbeddingStats returns statistics about embeddings in the database
func (r *Repository) GetEmbeddingStats(ctx context.Context) (*EmbeddingStats, error) {
	stats := &EmbeddingStats{}

	// Count documents with embeddings
	docCollection := r.mongodb.Collection("documents")
	docWithEmbeddings, err := docCollection.CountDocuments(ctx, bson.M{
		"embeddings": bson.M{"$exists": true, "$ne": nil, "$not": bson.M{"$size": 0}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count documents with embeddings: %w", err)
	}
	stats.DocumentsWithEmbeddings = docWithEmbeddings

	// Count total documents
	totalDocs, err := docCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total documents: %w", err)
	}
	stats.TotalDocuments = totalDocs

	// Count knowledge items with embeddings
	knowledgeCollection := r.mongodb.Collection("knowledge_items")
	knowledgeWithEmbeddings, err := knowledgeCollection.CountDocuments(ctx, bson.M{
		"embeddings": bson.M{"$exists": true, "$ne": nil, "$not": bson.M{"$size": 0}},
		"is_active":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count knowledge items with embeddings: %w", err)
	}
	stats.KnowledgeWithEmbeddings = knowledgeWithEmbeddings

	// Count total active knowledge items
	totalKnowledge, err := knowledgeCollection.CountDocuments(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, fmt.Errorf("failed to count total knowledge items: %w", err)
	}
	stats.TotalKnowledgeItems = totalKnowledge

	return stats, nil
}

// EmbeddingStats represents statistics about embeddings in the database
type EmbeddingStats struct {
	DocumentsWithEmbeddings int64 `json:"documents_with_embeddings"`
	TotalDocuments          int64 `json:"total_documents"`
	KnowledgeWithEmbeddings int64 `json:"knowledge_with_embeddings"`
	TotalKnowledgeItems     int64 `json:"total_knowledge_items"`
}

// DeleteDocumentEmbeddings removes embeddings from a document
func (r *Repository) DeleteDocumentEmbeddings(ctx context.Context, documentID primitive.ObjectID) error {
	collection := r.mongodb.Collection("documents")

	update := bson.M{
		"$unset": bson.M{
			"embeddings": "",
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
	if err != nil {
		return fmt.Errorf("failed to delete document embeddings: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found: %s", documentID.Hex())
	}

	return nil
}

// DeleteKnowledgeEmbeddings removes embeddings from a knowledge item
func (r *Repository) DeleteKnowledgeEmbeddings(ctx context.Context, knowledgeID primitive.ObjectID) error {
	collection := r.mongodb.Collection("knowledge_items")

	update := bson.M{
		"$unset": bson.M{
			"embeddings": "",
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": knowledgeID}, update)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge embeddings: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("knowledge item not found: %s", knowledgeID.Hex())
	}

	return nil
}
