package research

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

// MongoResearchRepository implements ResearchRepository using MongoDB
type MongoResearchRepository struct {
	db                      *mongo.Database
	researchResultsCollection *mongo.Collection
	policySuggestionsCollection *mongo.Collection
	currentEventsCollection   *mongo.Collection
	researchSourcesCollection *mongo.Collection
}

// NewMongoResearchRepository creates a new MongoDB research repository
func NewMongoResearchRepository(db *mongo.Database) *MongoResearchRepository {
	return &MongoResearchRepository{
		db:                        db,
		researchResultsCollection: db.Collection("research_results"),
		policySuggestionsCollection: db.Collection("policy_suggestions"),
		currentEventsCollection:   db.Collection("current_events"),
		researchSourcesCollection: db.Collection("research_sources"),
	}
}

// SaveResearchResult saves a research result to the database
func (r *MongoResearchRepository) SaveResearchResult(ctx context.Context, result *models.ResearchResult) error {
	if result.ID.IsZero() {
		result.ID = primitive.NewObjectID()
	}
	
	if result.GeneratedAt.IsZero() {
		result.GeneratedAt = time.Now()
	}
	
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	
	_, err := r.researchResultsCollection.InsertOne(ctx, result)
	if err != nil {
		return fmt.Errorf("failed to save research result: %w", err)
	}
	
	return nil
}

// GetResearchResult retrieves a research result by ID
func (r *MongoResearchRepository) GetResearchResult(ctx context.Context, id string) (*models.ResearchResult, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid research result ID: %w", err)
	}
	
	var result models.ResearchResult
	err = r.researchResultsCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("research result not found")
		}
		return nil, fmt.Errorf("failed to get research result: %w", err)
	}
	
	return &result, nil
}

// GetResearchResultsByDocument retrieves research results for a specific document
func (r *MongoResearchRepository) GetResearchResultsByDocument(ctx context.Context, documentID string) ([]models.ResearchResult, error) {
	objectID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	
	filter := bson.M{"document_id": objectID}
	opts := options.Find().SetSort(bson.D{{"generated_at", -1}})
	
	cursor, err := r.researchResultsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query research results: %w", err)
	}
	defer cursor.Close(ctx)
	
	var results []models.ResearchResult
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode research results: %w", err)
	}
	
	return results, nil
}

// SavePolicySuggestion saves a policy suggestion to the database
func (r *MongoResearchRepository) SavePolicySuggestion(ctx context.Context, suggestion *models.PolicySuggestion) error {
	if suggestion.ID.IsZero() {
		suggestion.ID = primitive.NewObjectID()
	}
	
	now := time.Now()
	if suggestion.CreatedAt.IsZero() {
		suggestion.CreatedAt = now
	}
	suggestion.UpdatedAt = now
	
	if suggestion.Status == "" {
		suggestion.Status = "draft"
	}
	
	_, err := r.policySuggestionsCollection.InsertOne(ctx, suggestion)
	if err != nil {
		return fmt.Errorf("failed to save policy suggestion: %w", err)
	}
	
	return nil
}

// GetPolicySuggestion retrieves a policy suggestion by ID
func (r *MongoResearchRepository) GetPolicySuggestion(ctx context.Context, id string) (*models.PolicySuggestion, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid policy suggestion ID: %w", err)
	}
	
	var suggestion models.PolicySuggestion
	err = r.policySuggestionsCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&suggestion)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("policy suggestion not found")
		}
		return nil, fmt.Errorf("failed to get policy suggestion: %w", err)
	}
	
	return &suggestion, nil
}

// GetPolicySuggestionsByCategory retrieves policy suggestions by category
func (r *MongoResearchRepository) GetPolicySuggestionsByCategory(ctx context.Context, category models.DocumentCategory) ([]models.PolicySuggestion, error) {
	filter := bson.M{"category": category}
	opts := options.Find().SetSort(bson.D{{"created_at", -1}})
	
	cursor, err := r.policySuggestionsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy suggestions: %w", err)
	}
	defer cursor.Close(ctx)
	
	var suggestions []models.PolicySuggestion
	if err = cursor.All(ctx, &suggestions); err != nil {
		return nil, fmt.Errorf("failed to decode policy suggestions: %w", err)
	}
	
	return suggestions, nil
}

// SaveCurrentEvent saves a current event to the database
func (r *MongoResearchRepository) SaveCurrentEvent(ctx context.Context, event *models.CurrentEvent) error {
	if event.ID.IsZero() {
		event.ID = primitive.NewObjectID()
	}
	
	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now
	
	// Check if event already exists by URL to avoid duplicates
	existing := r.currentEventsCollection.FindOne(ctx, bson.M{"url": event.URL})
	if existing.Err() == nil {
		// Update existing event
		update := bson.M{
			"$set": bson.M{
				"title":       event.Title,
				"description": event.Description,
				"content":     event.Content,
				"relevance":   event.Relevance,
				"tags":        event.Tags,
				"updated_at":  now,
			},
		}
		_, err := r.currentEventsCollection.UpdateOne(ctx, bson.M{"url": event.URL}, update)
		return err
	}
	
	_, err := r.currentEventsCollection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to save current event: %w", err)
	}
	
	return nil
}

// GetCurrentEvents retrieves current events with optional filters
func (r *MongoResearchRepository) GetCurrentEvents(ctx context.Context, filters CurrentEventFilters) ([]models.CurrentEvent, error) {
	filter := bson.M{}
	
	if filters.Category != nil {
		filter["category"] = *filters.Category
	}
	
	if len(filters.Tags) > 0 {
		filter["tags"] = bson.M{"$in": filters.Tags}
	}
	
	if filters.DateFrom != nil || filters.DateTo != nil {
		dateFilter := bson.M{}
		if filters.DateFrom != nil {
			dateFilter["$gte"] = *filters.DateFrom
		}
		if filters.DateTo != nil {
			dateFilter["$lte"] = *filters.DateTo
		}
		filter["published_at"] = dateFilter
	}
	
	if filters.MinRelevance != nil {
		filter["relevance"] = bson.M{"$gte": *filters.MinRelevance}
	}
	
	if filters.Language != nil {
		filter["language"] = *filters.Language
	}
	
	if filters.Source != nil {
		filter["source"] = *filters.Source
	}
	
	opts := options.Find().
		SetSort(bson.D{{"published_at", -1}}).
		SetLimit(int64(filters.Limit)).
		SetSkip(int64(filters.Offset))
	
	cursor, err := r.currentEventsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query current events: %w", err)
	}
	defer cursor.Close(ctx)
	
	var events []models.CurrentEvent
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode current events: %w", err)
	}
	
	return events, nil
}

// SaveResearchSource saves a research source to the database
func (r *MongoResearchRepository) SaveResearchSource(ctx context.Context, source *models.ResearchSource) error {
	if source.ID.IsZero() {
		source.ID = primitive.NewObjectID()
	}
	
	now := time.Now()
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}
	source.UpdatedAt = now
	
	// Check if source already exists by URL to avoid duplicates
	existing := r.researchSourcesCollection.FindOne(ctx, bson.M{"url": source.URL})
	if existing.Err() == nil {
		// Update existing source
		update := bson.M{
			"$set": bson.M{
				"title":       source.Title,
				"content":     source.Content,
				"summary":     source.Summary,
				"credibility": source.Credibility,
				"relevance":   source.Relevance,
				"keywords":    source.Keywords,
				"updated_at":  now,
			},
		}
		_, err := r.researchSourcesCollection.UpdateOne(ctx, bson.M{"url": source.URL}, update)
		return err
	}
	
	_, err := r.researchSourcesCollection.InsertOne(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to save research source: %w", err)
	}
	
	return nil
}

// GetResearchSources retrieves research sources with optional filters
func (r *MongoResearchRepository) GetResearchSources(ctx context.Context, filters ResearchSourceFilters) ([]models.ResearchSource, error) {
	filter := bson.M{}
	
	if filters.Type != nil {
		filter["type"] = *filters.Type
	}
	
	if filters.MinCredibility != nil {
		filter["credibility"] = bson.M{"$gte": *filters.MinCredibility}
	}
	
	if filters.MinRelevance != nil {
		filter["relevance"] = bson.M{"$gte": *filters.MinRelevance}
	}
	
	if filters.DateFrom != nil || filters.DateTo != nil {
		dateFilter := bson.M{}
		if filters.DateFrom != nil {
			dateFilter["$gte"] = *filters.DateFrom
		}
		if filters.DateTo != nil {
			dateFilter["$lte"] = *filters.DateTo
		}
		filter["published_at"] = dateFilter
	}
	
	if len(filters.Keywords) > 0 {
		filter["keywords"] = bson.M{"$in": filters.Keywords}
	}
	
	if filters.Language != nil {
		filter["language"] = *filters.Language
	}
	
	opts := options.Find().
		SetSort(bson.D{{"credibility", -1}, {"relevance", -1}}).
		SetLimit(int64(filters.Limit)).
		SetSkip(int64(filters.Offset))
	
	cursor, err := r.researchSourcesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query research sources: %w", err)
	}
	defer cursor.Close(ctx)
	
	var sources []models.ResearchSource
	if err = cursor.All(ctx, &sources); err != nil {
		return nil, fmt.Errorf("failed to decode research sources: %w", err)
	}
	
	return sources, nil
}

// UpdatePolicySuggestionStatus updates the status of a policy suggestion
func (r *MongoResearchRepository) UpdatePolicySuggestionStatus(ctx context.Context, id string, status string, reviewNotes *string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid policy suggestion ID: %w", err)
	}
	
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	
	if reviewNotes != nil {
		update["$set"].(bson.M)["review_notes"] = *reviewNotes
	}
	
	if status == "approved" {
		update["$set"].(bson.M)["approved_at"] = time.Now()
	}
	
	result, err := r.policySuggestionsCollection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return fmt.Errorf("failed to update policy suggestion status: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("policy suggestion not found")
	}
	
	return nil
}

// CreateIndexes creates the necessary indexes for research collections
func (r *MongoResearchRepository) CreateIndexes(ctx context.Context) error {
	// Research results indexes
	researchIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"document_id", 1}}},
		{Keys: bson.D{{"status", 1}}},
		{Keys: bson.D{{"generated_at", -1}}},
		{Keys: bson.D{{"confidence", -1}}},
		{Keys: bson.D{{"research_query", "text"}}},
	}
	
	_, err := r.researchResultsCollection.Indexes().CreateMany(ctx, researchIndexes)
	if err != nil {
		return fmt.Errorf("failed to create research results indexes: %w", err)
	}
	
	// Policy suggestions indexes
	policyIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"category", 1}}},
		{Keys: bson.D{{"priority", 1}}},
		{Keys: bson.D{{"status", 1}}},
		{Keys: bson.D{{"created_at", -1}}},
		{Keys: bson.D{{"confidence", -1}}},
		{Keys: bson.D{{"created_by", 1}}},
		{Keys: bson.D{{"tags", 1}}},
		{Keys: bson.D{{"title", "text"}, {"description", "text"}}},
	}
	
	_, err = r.policySuggestionsCollection.Indexes().CreateMany(ctx, policyIndexes)
	if err != nil {
		return fmt.Errorf("failed to create policy suggestions indexes: %w", err)
	}
	
	// Current events indexes
	eventIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"category", 1}}},
		{Keys: bson.D{{"tags", 1}}},
		{Keys: bson.D{{"published_at", -1}}},
		{Keys: bson.D{{"relevance", -1}}},
		{Keys: bson.D{{"source", 1}}},
		{Keys: bson.D{{"language", 1}}},
		{Keys: bson.D{{"url", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"title", "text"}, {"description", "text"}, {"content", "text"}}},
	}
	
	_, err = r.currentEventsCollection.Indexes().CreateMany(ctx, eventIndexes)
	if err != nil {
		return fmt.Errorf("failed to create current events indexes: %w", err)
	}
	
	// Research sources indexes
	sourceIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"type", 1}}},
		{Keys: bson.D{{"credibility", -1}}},
		{Keys: bson.D{{"relevance", -1}}},
		{Keys: bson.D{{"published_at", -1}}},
		{Keys: bson.D{{"keywords", 1}}},
		{Keys: bson.D{{"language", 1}}},
		{Keys: bson.D{{"url", 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{"title", "text"}, {"content", "text"}, {"summary", "text"}}},
	}
	
	_, err = r.researchSourcesCollection.Indexes().CreateMany(ctx, sourceIndexes)
	if err != nil {
		return fmt.Errorf("failed to create research sources indexes: %w", err)
	}
	
	return nil
}