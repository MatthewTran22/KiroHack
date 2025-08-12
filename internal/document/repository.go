package document

import (
	"context"
	"fmt"
	"time"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles document data access operations
type Repository struct {
	collection *mongo.Collection
}

// NewRepository creates a new document repository
func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		collection: db.Collection("documents"),
	}
}

// CreateIndexes creates necessary indexes for the documents collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "content", Value: "text"},
				{Key: "metadata.title", Value: "text"},
				{Key: "metadata.tags", Value: "text"},
			},
			Options: options.Index().SetName("text_search_index"),
		},
		{
			Keys:    bson.D{{Key: "uploaded_by", Value: 1}},
			Options: options.Index().SetName("uploaded_by_index"),
		},
		{
			Keys:    bson.D{{Key: "uploaded_at", Value: -1}},
			Options: options.Index().SetName("uploaded_at_index"),
		},
		{
			Keys:    bson.D{{Key: "processing_status", Value: 1}},
			Options: options.Index().SetName("processing_status_index"),
		},
		{
			Keys:    bson.D{{Key: "metadata.category", Value: 1}},
			Options: options.Index().SetName("category_index"),
		},
		{
			Keys:    bson.D{{Key: "metadata.tags", Value: 1}},
			Options: options.Index().SetName("tags_index"),
		},
		{
			Keys:    bson.D{{Key: "classification.level", Value: 1}},
			Options: options.Index().SetName("classification_index"),
		},
	}

	for _, index := range indexes {
		_, err := r.collection.Indexes().CreateOne(ctx, index)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", *index.Options.Name, err)
		}
	}

	return nil
}

// Create inserts a new document
func (r *Repository) Create(ctx context.Context, doc *models.Document) error {
	if doc.ID.IsZero() {
		doc.ID = primitive.NewObjectID()
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// GetByID retrieves a document by its ID
func (r *Repository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Document, error) {
	var doc models.Document
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &doc, nil
}

// Update updates a document
func (r *Repository) Update(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// Delete deletes a document by ID
func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// SearchFilter represents search criteria for documents
type SearchFilter struct {
	Query          string                  `json:"query"`
	Category       models.DocumentCategory `json:"category"`
	Tags           []string                `json:"tags"`
	UploadedBy     *primitive.ObjectID     `json:"uploaded_by"`
	Classification string                  `json:"classification"`
	Status         models.ProcessingStatus `json:"status"`
	DateFrom       *time.Time              `json:"date_from"`
	DateTo         *time.Time              `json:"date_to"`
	Limit          int                     `json:"limit"`
	Skip           int                     `json:"skip"`
}

// Search searches for documents based on criteria
func (r *Repository) Search(ctx context.Context, filter SearchFilter) ([]*models.Document, int64, error) {
	// Build the query
	query := bson.M{}

	// Text search
	if filter.Query != "" {
		query["$text"] = bson.M{"$search": filter.Query}
	}

	// Category filter
	if filter.Category != "" {
		query["metadata.category"] = filter.Category
	}

	// Tags filter
	if len(filter.Tags) > 0 {
		query["metadata.tags"] = bson.M{"$in": filter.Tags}
	}

	// Uploaded by filter
	if filter.UploadedBy != nil {
		query["uploaded_by"] = *filter.UploadedBy
	}

	// Classification filter
	if filter.Classification != "" {
		query["classification.level"] = filter.Classification
	}

	// Status filter
	if filter.Status != "" {
		query["processing_status"] = filter.Status
	}

	// Date range filter
	if filter.DateFrom != nil || filter.DateTo != nil {
		dateQuery := bson.M{}
		if filter.DateFrom != nil {
			dateQuery["$gte"] = *filter.DateFrom
		}
		if filter.DateTo != nil {
			dateQuery["$lte"] = *filter.DateTo
		}
		query["uploaded_at"] = dateQuery
	}

	// Count total documents matching the query
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	// Set default limit if not specified
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	// Build find options
	findOptions := options.Find()
	findOptions.SetLimit(int64(filter.Limit))
	findOptions.SetSkip(int64(filter.Skip))

	// If text search is used, sort by text score, otherwise sort by upload date
	if filter.Query != "" {
		findOptions.SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}})
	} else {
		findOptions.SetSort(bson.D{{Key: "uploaded_at", Value: -1}}) // Sort by upload date, newest first
	}

	// Execute the query
	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search documents: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var documents []*models.Document
	for cursor.Next(ctx) {
		var doc models.Document
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}

	return documents, total, nil
}

// GetByStatus retrieves documents by processing status
func (r *Repository) GetByStatus(ctx context.Context, status models.ProcessingStatus, limit int) ([]*models.Document, error) {
	if limit <= 0 {
		limit = 50
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.D{{Key: "uploaded_at", Value: 1}}) // Oldest first for processing queue

	cursor, err := r.collection.Find(ctx, bson.M{"processing_status": status}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents by status: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*models.Document
	for cursor.Next(ctx) {
		var doc models.Document
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return documents, nil
}

// GetByUser retrieves documents uploaded by a specific user
func (r *Repository) GetByUser(ctx context.Context, userID primitive.ObjectID, limit, skip int) ([]*models.Document, int64, error) {
	if limit <= 0 {
		limit = 20
	}

	query := bson.M{"uploaded_by": userID}

	// Count total documents for the user
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user documents: %w", err)
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSkip(int64(skip))
	findOptions.SetSort(bson.D{{Key: "uploaded_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []*models.Document
	for cursor.Next(ctx) {
		var doc models.Document
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, fmt.Errorf("failed to decode document: %w", err)
		}
		documents = append(documents, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}

	return documents, total, nil
}

// UpdateProcessingStatus updates the processing status of a document
func (r *Repository) UpdateProcessingStatus(ctx context.Context, id primitive.ObjectID, status models.ProcessingStatus, errorMsg string) error {
	update := bson.M{
		"$set": bson.M{
			"processing_status": status,
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["processing_error"] = errorMsg
	} else {
		update["$unset"] = bson.M{"processing_error": ""}
	}

	return r.Update(ctx, id, update)
}

// GetStatistics returns document statistics
func (r *Repository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total documents
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total documents: %w", err)
	}
	stats["total"] = total

	// Documents by status
	statusPipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$processing_status",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, statusPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate status statistics: %w", err)
	}
	defer cursor.Close(ctx)

	statusStats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode status statistics: %w", err)
		}
		statusStats[result.ID] = result.Count
	}
	stats["by_status"] = statusStats

	// Documents by category
	categoryPipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$metadata.category",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err = r.collection.Aggregate(ctx, categoryPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate category statistics: %w", err)
	}
	defer cursor.Close(ctx)

	categoryStats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode category statistics: %w", err)
		}
		categoryStats[result.ID] = result.Count
	}
	stats["by_category"] = categoryStats

	return stats, nil
}
