package knowledge

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

// Repository handles knowledge data access operations
type Repository struct {
	collection *mongo.Collection
}

// NewRepository creates a new knowledge repository
func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		collection: db.Collection("knowledge_items"),
	}
}

// CreateIndexes creates necessary indexes for the knowledge_items collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "title", Value: "text"},
				{Key: "content", Value: "text"},
				{Key: "summary", Value: "text"},
				{Key: "keywords", Value: "text"},
			},
			Options: options.Index().SetName("text_search_index"),
		},
		{
			Keys:    bson.D{{Key: "type", Value: 1}},
			Options: options.Index().SetName("type_index"),
		},
		{
			Keys:    bson.D{{Key: "category", Value: 1}},
			Options: options.Index().SetName("category_index"),
		},
		{
			Keys:    bson.D{{Key: "tags", Value: 1}},
			Options: options.Index().SetName("tags_index"),
		},
		{
			Keys:    bson.D{{Key: "keywords", Value: 1}},
			Options: options.Index().SetName("keywords_index"),
		},
		{
			Keys:    bson.D{{Key: "created_by", Value: 1}},
			Options: options.Index().SetName("created_by_index"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("created_at_index"),
		},
		{
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetName("updated_at_index"),
		},
		{
			Keys:    bson.D{{Key: "is_active", Value: 1}},
			Options: options.Index().SetName("is_active_index"),
		},
		{
			Keys:    bson.D{{Key: "confidence", Value: -1}},
			Options: options.Index().SetName("confidence_index"),
		},
		{
			Keys:    bson.D{{Key: "validation.is_validated", Value: 1}},
			Options: options.Index().SetName("validation_index"),
		},
		{
			Keys:    bson.D{{Key: "validation.expires_at", Value: 1}},
			Options: options.Index().SetName("expiration_index"),
		},
		{
			Keys:    bson.D{{Key: "source.type", Value: 1}},
			Options: options.Index().SetName("source_type_index"),
		},
		{
			Keys:    bson.D{{Key: "source.source_id", Value: 1}},
			Options: options.Index().SetName("source_id_index"),
		},
		{
			Keys:    bson.D{{Key: "relationships.target_id", Value: 1}},
			Options: options.Index().SetName("relationships_target_index"),
		},
		{
			Keys:    bson.D{{Key: "relationships.type", Value: 1}},
			Options: options.Index().SetName("relationships_type_index"),
		},
		{
			Keys:    bson.D{{Key: "usage.access_count", Value: -1}},
			Options: options.Index().SetName("usage_count_index"),
		},
		{
			Keys:    bson.D{{Key: "usage.last_accessed", Value: -1}},
			Options: options.Index().SetName("last_accessed_index"),
		},
		{
			Keys:    bson.D{{Key: "usage.effectiveness_score", Value: -1}},
			Options: options.Index().SetName("effectiveness_index"),
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

// Create inserts a new knowledge item
func (r *Repository) Create(ctx context.Context, item *models.KnowledgeItem) error {
	if item.ID.IsZero() {
		item.ID = primitive.NewObjectID()
	}

	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Version = 1
	item.IsActive = true

	if err := item.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	_, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		return fmt.Errorf("failed to create knowledge item: %w", err)
	}

	return nil
}

// GetByID retrieves a knowledge item by its ID
func (r *Repository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.KnowledgeItem, error) {
	var item models.KnowledgeItem
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "is_active": true}).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("knowledge item not found")
		}
		return nil, fmt.Errorf("failed to get knowledge item: %w", err)
	}

	return &item, nil
}

// Update updates a knowledge item
func (r *Repository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	// Add updated timestamp and increment version
	updates["updated_at"] = time.Now()
	updates["$inc"] = bson.M{"version": 1}

	update := bson.M{"$set": updates}
	if inc, exists := updates["$inc"]; exists {
		update["$inc"] = inc
		delete(updates, "$inc")
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id, "is_active": true}, update)
	if err != nil {
		return fmt.Errorf("failed to update knowledge item: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("knowledge item not found")
	}

	return nil
}

// Delete soft deletes a knowledge item by setting is_active to false
func (r *Repository) Delete(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{
		"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		},
		"$inc": bson.M{"version": 1},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id, "is_active": true}, update)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge item: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("knowledge item not found")
	}

	return nil
}

// SearchFilter represents search criteria for knowledge items
type SearchFilter struct {
	Query              string                    `json:"query"`
	Type               models.KnowledgeType      `json:"type"`
	Category           string                    `json:"category"`
	Tags               []string                  `json:"tags"`
	Keywords           []string                  `json:"keywords"`
	CreatedBy          *primitive.ObjectID       `json:"created_by"`
	SourceType         string                    `json:"source_type"`
	SourceID           *primitive.ObjectID       `json:"source_id"`
	MinConfidence      float64                   `json:"min_confidence"`
	MaxConfidence      float64                   `json:"max_confidence"`
	IsValidated        *bool                     `json:"is_validated"`
	NotExpired         bool                      `json:"not_expired"`
	MinUsageCount      int64                     `json:"min_usage_count"`
	MinEffectiveness   float64                   `json:"min_effectiveness"`
	RelationshipType   models.RelationshipType   `json:"relationship_type"`
	RelationshipTarget *primitive.ObjectID       `json:"relationship_target"`
	DateFrom           *time.Time                `json:"date_from"`
	DateTo             *time.Time                `json:"date_to"`
	Limit              int                       `json:"limit"`
	Skip               int                       `json:"skip"`
	SortBy             string                    `json:"sort_by"` // "created_at", "updated_at", "confidence", "usage_count", "effectiveness"
	SortOrder          int                       `json:"sort_order"` // 1 for ascending, -1 for descending
}

// Search searches for knowledge items based on criteria
func (r *Repository) Search(ctx context.Context, filter SearchFilter) ([]*models.KnowledgeItem, int64, error) {
	// Build the query
	query := bson.M{"is_active": true}

	// Text search
	if filter.Query != "" {
		query["$text"] = bson.M{"$search": filter.Query}
	}

	// Type filter
	if filter.Type != "" {
		query["type"] = filter.Type
	}

	// Category filter
	if filter.Category != "" {
		query["category"] = filter.Category
	}

	// Tags filter
	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}

	// Keywords filter
	if len(filter.Keywords) > 0 {
		query["keywords"] = bson.M{"$in": filter.Keywords}
	}

	// Created by filter
	if filter.CreatedBy != nil {
		query["created_by"] = *filter.CreatedBy
	}

	// Source filters
	if filter.SourceType != "" {
		query["source.type"] = filter.SourceType
	}
	if filter.SourceID != nil {
		query["source.source_id"] = *filter.SourceID
	}

	// Confidence range filter
	if filter.MinConfidence > 0 || filter.MaxConfidence > 0 {
		confidenceQuery := bson.M{}
		if filter.MinConfidence > 0 {
			confidenceQuery["$gte"] = filter.MinConfidence
		}
		if filter.MaxConfidence > 0 {
			confidenceQuery["$lte"] = filter.MaxConfidence
		}
		query["confidence"] = confidenceQuery
	}

	// Validation filter
	if filter.IsValidated != nil {
		query["validation.is_validated"] = *filter.IsValidated
	}

	// Not expired filter
	if filter.NotExpired {
		query["$or"] = []bson.M{
			{"validation.expires_at": bson.M{"$exists": false}},
			{"validation.expires_at": nil},
			{"validation.expires_at": bson.M{"$gt": time.Now()}},
		}
	}

	// Usage filters
	if filter.MinUsageCount > 0 {
		query["usage.access_count"] = bson.M{"$gte": filter.MinUsageCount}
	}
	if filter.MinEffectiveness > 0 {
		query["usage.effectiveness_score"] = bson.M{"$gte": filter.MinEffectiveness}
	}

	// Relationship filters
	if filter.RelationshipType != "" || filter.RelationshipTarget != nil {
		relationshipQuery := bson.M{}
		if filter.RelationshipType != "" {
			relationshipQuery["relationships.type"] = filter.RelationshipType
		}
		if filter.RelationshipTarget != nil {
			relationshipQuery["relationships.target_id"] = *filter.RelationshipTarget
		}
		query = bson.M{"$and": []bson.M{query, relationshipQuery}}
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
		query["created_at"] = dateQuery
	}

	// Count total items matching the query
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count knowledge items: %w", err)
	}

	// Set default limit if not specified
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	// Build find options
	findOptions := options.Find()
	findOptions.SetLimit(int64(filter.Limit))
	findOptions.SetSkip(int64(filter.Skip))

	// Set sort order
	sortField := "created_at"
	sortOrder := -1 // Default to descending

	if filter.SortBy != "" {
		switch filter.SortBy {
		case "created_at", "updated_at", "confidence", "usage.access_count", "usage.effectiveness_score":
			sortField = filter.SortBy
		}
	}

	if filter.SortOrder != 0 {
		sortOrder = filter.SortOrder
	}

	// If text search is used, sort by text score first, then by specified field
	if filter.Query != "" {
		findOptions.SetSort(bson.D{
			{Key: "score", Value: bson.M{"$meta": "textScore"}},
			{Key: sortField, Value: sortOrder},
		})
	} else {
		findOptions.SetSort(bson.D{{Key: sortField, Value: sortOrder}})
	}

	// Execute the query
	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search knowledge items: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode results
	var items []*models.KnowledgeItem
	for cursor.Next(ctx) {
		var item models.KnowledgeItem
		if err := cursor.Decode(&item); err != nil {
			return nil, 0, fmt.Errorf("failed to decode knowledge item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}

	return items, total, nil
}

// GetByType retrieves knowledge items by type
func (r *Repository) GetByType(ctx context.Context, knowledgeType models.KnowledgeType, limit int) ([]*models.KnowledgeItem, error) {
	if limit <= 0 {
		limit = 50
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.D{{Key: "confidence", Value: -1}}) // Sort by confidence, highest first

	cursor, err := r.collection.Find(ctx, bson.M{"type": knowledgeType, "is_active": true}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge items by type: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*models.KnowledgeItem
	for cursor.Next(ctx) {
		var item models.KnowledgeItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode knowledge item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return items, nil
}

// GetBySource retrieves knowledge items by source
func (r *Repository) GetBySource(ctx context.Context, sourceType string, sourceID primitive.ObjectID, limit int) ([]*models.KnowledgeItem, error) {
	if limit <= 0 {
		limit = 50
	}

	query := bson.M{
		"source.type":      sourceType,
		"source.source_id": sourceID,
		"is_active":        true,
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}}) // Sort by creation date, newest first

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge items by source: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*models.KnowledgeItem
	for cursor.Next(ctx) {
		var item models.KnowledgeItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode knowledge item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return items, nil
}

// GetRelatedItems retrieves knowledge items related to a specific item
func (r *Repository) GetRelatedItems(ctx context.Context, itemID primitive.ObjectID, relationshipType models.RelationshipType, limit int) ([]*models.KnowledgeItem, error) {
	if limit <= 0 {
		limit = 20
	}

	// Build query to find items that have relationships to the specified item
	query := bson.M{
		"relationships": bson.M{
			"$elemMatch": bson.M{
				"target_id": itemID,
			},
		},
		"is_active": true,
	}

	// Add relationship type filter if specified
	if relationshipType != "" {
		query["relationships"].(bson.M)["$elemMatch"].(bson.M)["type"] = relationshipType
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.D{{Key: "relationships.strength", Value: -1}}) // Sort by relationship strength

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get related knowledge items: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*models.KnowledgeItem
	for cursor.Next(ctx) {
		var item models.KnowledgeItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode knowledge item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return items, nil
}

// GetExpiredItems retrieves knowledge items that have expired
func (r *Repository) GetExpiredItems(ctx context.Context, limit int) ([]*models.KnowledgeItem, error) {
	if limit <= 0 {
		limit = 100
	}

	query := bson.M{
		"validation.expires_at": bson.M{
			"$exists": true,
			"$ne":     nil,
			"$lt":     time.Now(),
		},
		"is_active": true,
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(limit))
	findOptions.SetSort(bson.D{{Key: "validation.expires_at", Value: 1}}) // Sort by expiration date, oldest first

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired knowledge items: %w", err)
	}
	defer cursor.Close(ctx)

	var items []*models.KnowledgeItem
	for cursor.Next(ctx) {
		var item models.KnowledgeItem
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode knowledge item: %w", err)
		}
		items = append(items, &item)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return items, nil
}

// UpdateUsage updates the usage statistics for a knowledge item
func (r *Repository) UpdateUsage(ctx context.Context, id primitive.ObjectID, context string) error {
	now := time.Now()
	
	// First, get the current item to check existing contexts
	var item models.KnowledgeItem
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "is_active": true}).Decode(&item)
	if err != nil {
		return fmt.Errorf("failed to find knowledge item: %w", err)
	}

	// Check if context already exists
	contextExists := false
	for _, ctx := range item.Usage.UsageContexts {
		if ctx == context {
			contextExists = true
			break
		}
	}

	update := bson.M{
		"$inc": bson.M{
			"usage.access_count": 1,
		},
		"$set": bson.M{
			"usage.last_accessed": now,
		},
	}

	// Add context if it doesn't exist
	if !contextExists {
		update["$addToSet"] = bson.M{
			"usage.usage_contexts": context,
		}
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id, "is_active": true}, update)
	if err != nil {
		return fmt.Errorf("failed to update usage: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("knowledge item not found")
	}

	return nil
}

// AddRelationship adds a relationship between two knowledge items
func (r *Repository) AddRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType, strength float64, relationshipContext string) error {
	relationship := models.KnowledgeRelationship{
		Type:      relType,
		TargetID:  targetID,
		Strength:  strength,
		Context:   relationshipContext,
		CreatedAt: time.Now(),
	}

	update := bson.M{
		"$push": bson.M{
			"relationships": relationship,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": sourceID, "is_active": true}, update)
	if err != nil {
		return fmt.Errorf("failed to add relationship: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("source knowledge item not found")
	}

	return nil
}

// RemoveRelationship removes a relationship between two knowledge items
func (r *Repository) RemoveRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType) error {
	update := bson.M{
		"$pull": bson.M{
			"relationships": bson.M{
				"target_id": targetID,
				"type":      relType,
			},
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": sourceID, "is_active": true}, update)
	if err != nil {
		return fmt.Errorf("failed to remove relationship: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("source knowledge item not found")
	}

	return nil
}

// GetStatistics returns knowledge statistics
func (r *Repository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total active knowledge items
	total, err := r.collection.CountDocuments(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, fmt.Errorf("failed to count total knowledge items: %w", err)
	}
	stats["total"] = total

	// Knowledge items by type
	typePipeline := []bson.M{
		{"$match": bson.M{"is_active": true}},
		{"$group": bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate type statistics: %w", err)
	}
	defer cursor.Close(ctx)

	typeStats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode type statistics: %w", err)
		}
		typeStats[result.ID] = result.Count
	}
	stats["by_type"] = typeStats

	// Knowledge items by category
	categoryPipeline := []bson.M{
		{"$match": bson.M{"is_active": true}},
		{"$group": bson.M{
			"_id":   "$category",
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

	// Validation statistics
	validatedCount, err := r.collection.CountDocuments(ctx, bson.M{
		"is_active":               true,
		"validation.is_validated": true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count validated items: %w", err)
	}

	expiredCount, err := r.collection.CountDocuments(ctx, bson.M{
		"is_active": true,
		"validation.expires_at": bson.M{
			"$exists": true,
			"$ne":     nil,
			"$lt":     time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count expired items: %w", err)
	}

	stats["validation"] = map[string]int64{
		"validated": validatedCount,
		"expired":   expiredCount,
	}

	// Average confidence score
	confidencePipeline := []bson.M{
		{"$match": bson.M{"is_active": true}},
		{"$group": bson.M{
			"_id":        nil,
			"avg_confidence": bson.M{"$avg": "$confidence"},
			"min_confidence": bson.M{"$min": "$confidence"},
			"max_confidence": bson.M{"$max": "$confidence"},
		}},
	}

	cursor, err = r.collection.Aggregate(ctx, confidencePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate confidence statistics: %w", err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			AvgConfidence float64 `bson:"avg_confidence"`
			MinConfidence float64 `bson:"min_confidence"`
			MaxConfidence float64 `bson:"max_confidence"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode confidence statistics: %w", err)
		}
		stats["confidence"] = map[string]float64{
			"average": result.AvgConfidence,
			"minimum": result.MinConfidence,
			"maximum": result.MaxConfidence,
		}
	}

	return stats, nil
}