package knowledge

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRepository_CreateAndGet(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test item
	item := &models.KnowledgeItem{
		Content:    "Test repository content",
		Type:       models.KnowledgeTypeFact,
		Title:      "Test Repository Item",
		Category:   "Test Category",
		Tags:       []string{"test", "repository"},
		Keywords:   []string{"test", "repository"},
		Source: models.KnowledgeSource{
			Type:        "manual",
			SourceID:    primitive.NewObjectID(),
			Reference:   "Test Reference",
			Reliability: 0.9,
		},
		Confidence: 0.8,
		Validation: models.KnowledgeValidation{
			IsValidated: false,
		},
		Usage: models.KnowledgeUsage{
			AccessCount:        0,
			UsageContexts:      []string{},
			EffectivenessScore: 0.0,
		},
		CreatedBy:     createdBy,
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	// Test create
	err := repo.Create(ctx, item)
	if err != nil {
		t.Errorf("Create() error = %v", err)
		return
	}

	if item.ID.IsZero() {
		t.Errorf("Create() should set ID")
	}

	if item.Version != 1 {
		t.Errorf("Create() version = %d, want 1", item.Version)
	}

	if !item.IsActive {
		t.Errorf("Create() IsActive = false, want true")
	}

	// Test get
	retrievedItem, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Errorf("GetByID() error = %v", err)
		return
	}

	if retrievedItem.ID != item.ID {
		t.Errorf("GetByID() ID = %v, want %v", retrievedItem.ID, item.ID)
	}

	if retrievedItem.Title != item.Title {
		t.Errorf("GetByID() Title = %v, want %v", retrievedItem.Title, item.Title)
	}
}

func TestRepository_Update(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test item
	item := &models.KnowledgeItem{
		Content:       "Original content",
		Type:          models.KnowledgeTypeFact,
		Title:         "Original Title",
		Category:      "Test Category",
		Tags:          []string{"test"},
		Keywords:      []string{"test"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
		Confidence:    0.7,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	err := repo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	originalVersion := item.Version

	// Test update
	updates := map[string]interface{}{
		"title":      "Updated Title",
		"content":    "Updated content",
		"confidence": 0.9,
	}

	err = repo.Update(ctx, item.ID, updates)
	if err != nil {
		t.Errorf("Update() error = %v", err)
		return
	}

	// Verify updates
	updatedItem, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Errorf("Failed to get updated item: %v", err)
		return
	}

	if updatedItem.Title != "Updated Title" {
		t.Errorf("Update() Title = %v, want 'Updated Title'", updatedItem.Title)
	}

	if updatedItem.Content != "Updated content" {
		t.Errorf("Update() Content = %v, want 'Updated content'", updatedItem.Content)
	}

	if updatedItem.Confidence != 0.9 {
		t.Errorf("Update() Confidence = %v, want 0.9", updatedItem.Confidence)
	}

	if updatedItem.Version != originalVersion+1 {
		t.Errorf("Update() Version = %v, want %v", updatedItem.Version, originalVersion+1)
	}
}

func TestRepository_Delete(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test item
	item := &models.KnowledgeItem{
		Content:       "Test content",
		Type:          models.KnowledgeTypeFact,
		Title:         "Test Title",
		Category:      "Test Category",
		Tags:          []string{"test"},
		Keywords:      []string{"test"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
		Confidence:    0.8,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	err := repo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	originalVersion := item.Version

	// Test delete (soft delete)
	err = repo.Delete(ctx, item.ID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
		return
	}

	// Verify item is soft deleted
	_, err = repo.GetByID(ctx, item.ID)
	if err == nil {
		t.Errorf("Delete() item should be soft deleted and not retrievable")
	}

	// Verify item still exists but is inactive
	if !item.IsActive {
		// Item should be marked as inactive
		if item.Version != originalVersion+1 {
			t.Errorf("Delete() should increment version, got %d, want %d", item.Version, originalVersion+1)
		}
	}
}

func TestRepository_Search(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items with different properties
	items := []*models.KnowledgeItem{
		{
			Content:       "Security regulation content",
			Type:          models.KnowledgeTypeRegulation,
			Title:         "Security Regulation",
			Category:      "Security",
			Tags:          []string{"security", "regulation"},
			Keywords:      []string{"security", "regulation"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence:    0.9,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Technology best practice content",
			Type:          models.KnowledgeTypeBestPractice,
			Title:         "Technology Best Practice",
			Category:      "Technology",
			Tags:          []string{"technology", "best-practice"},
			Keywords:      []string{"technology", "practice"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.8},
			Confidence:    0.7,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Policy procedure content",
			Type:          models.KnowledgeTypeProcedure,
			Title:         "Policy Procedure",
			Category:      "Policy",
			Tags:          []string{"policy", "procedure"},
			Keywords:      []string{"policy", "procedure"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.85},
			Confidence:    0.8,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	// Create all test items
	for _, item := range items {
		err := repo.Create(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}
	}

	tests := []struct {
		name          string
		filter        SearchFilter
		expectedCount int
	}{
		{
			name:          "search all items",
			filter:        SearchFilter{Limit: 10},
			expectedCount: 3,
		},
		{
			name:          "search by type",
			filter:        SearchFilter{Type: models.KnowledgeTypeRegulation, Limit: 10},
			expectedCount: 1,
		},
		{
			name:          "search by category",
			filter:        SearchFilter{Category: "Security", Limit: 10},
			expectedCount: 1,
		},
		{
			name:          "search by minimum confidence",
			filter:        SearchFilter{MinConfidence: 0.8, Limit: 10},
			expectedCount: 2,
		},
		{
			name:          "search with limit",
			filter:        SearchFilter{Limit: 2},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := repo.Search(ctx, tt.filter)
			if err != nil {
				t.Errorf("Search() error = %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Search() results count = %d, want %d", len(results), tt.expectedCount)
			}

			if total != int64(tt.expectedCount) {
				t.Errorf("Search() total = %d, want %d", total, tt.expectedCount)
			}

			// Verify all results are active
			for _, result := range results {
				if !result.IsActive {
					t.Errorf("Search() should only return active items")
				}
			}
		})
	}
}

func TestRepository_GetByType(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create items of different types
	types := []models.KnowledgeType{
		models.KnowledgeTypeFact,
		models.KnowledgeTypeRule,
		models.KnowledgeTypeFact, // Another fact
	}

	for i, knowledgeType := range types {
		item := &models.KnowledgeItem{
			Content:       fmt.Sprintf("Content %d", i),
			Type:          knowledgeType,
			Title:         fmt.Sprintf("Title %d", i),
			Category:      "Test Category",
			Tags:          []string{"test"},
			Keywords:      []string{"test"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence:    0.8,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		}

		err := repo.Create(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create test item %d: %v", i, err)
		}
	}

	// Test getting facts
	facts, err := repo.GetByType(ctx, models.KnowledgeTypeFact, 10)
	if err != nil {
		t.Errorf("GetByType() error = %v", err)
		return
	}

	if len(facts) != 2 {
		t.Errorf("GetByType() facts count = %d, want 2", len(facts))
	}

	for _, fact := range facts {
		if fact.Type != models.KnowledgeTypeFact {
			t.Errorf("GetByType() returned wrong type = %v, want %v", fact.Type, models.KnowledgeTypeFact)
		}
	}

	// Test getting rules
	rules, err := repo.GetByType(ctx, models.KnowledgeTypeRule, 10)
	if err != nil {
		t.Errorf("GetByType() error = %v", err)
		return
	}

	if len(rules) != 1 {
		t.Errorf("GetByType() rules count = %d, want 1", len(rules))
	}
}

func TestRepository_UpdateUsage(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test item
	item := &models.KnowledgeItem{
		Content:       "Test content",
		Type:          models.KnowledgeTypeFact,
		Title:         "Test Title",
		Category:      "Test Category",
		Tags:          []string{"test"},
		Keywords:      []string{"test"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
		Confidence:    0.8,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	err := repo.Create(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	originalAccessCount := item.Usage.AccessCount

	// Test updating usage
	err = repo.UpdateUsage(ctx, item.ID, "test_context")
	if err != nil {
		t.Errorf("UpdateUsage() error = %v", err)
		return
	}

	// Verify usage was updated
	updatedItem, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Errorf("Failed to get updated item: %v", err)
		return
	}

	if updatedItem.Usage.AccessCount != originalAccessCount+1 {
		t.Errorf("UpdateUsage() AccessCount = %d, want %d", updatedItem.Usage.AccessCount, originalAccessCount+1)
	}

	if updatedItem.Usage.LastAccessed == nil {
		t.Errorf("UpdateUsage() LastAccessed should not be nil")
	}

	// Check context was added
	found := false
	for _, ctx := range updatedItem.Usage.UsageContexts {
		if ctx == "test_context" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("UpdateUsage() should add context to UsageContexts")
	}

	// Test adding same context again (should not duplicate)
	err = repo.UpdateUsage(ctx, item.ID, "test_context")
	if err != nil {
		t.Errorf("UpdateUsage() error on duplicate context = %v", err)
		return
	}

	updatedItem2, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Errorf("Failed to get updated item: %v", err)
		return
	}

	// Count occurrences of test_context
	contextCount := 0
	for _, ctx := range updatedItem2.Usage.UsageContexts {
		if ctx == "test_context" {
			contextCount++
		}
	}

	if contextCount != 1 {
		t.Errorf("UpdateUsage() should not duplicate contexts, found %d occurrences", contextCount)
	}
}

func TestRepository_AddRemoveRelationship(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items
	item1 := &models.KnowledgeItem{
		Content:       "Item 1 content",
		Type:          models.KnowledgeTypeFact,
		Title:         "Item 1",
		Category:      "Test Category",
		Tags:          []string{"test"},
		Keywords:      []string{"test"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
		Confidence:    0.8,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	item2 := &models.KnowledgeItem{
		Content:       "Item 2 content",
		Type:          models.KnowledgeTypeRule,
		Title:         "Item 2",
		Category:      "Test Category",
		Tags:          []string{"test"},
		Keywords:      []string{"test"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
		Confidence:    0.8,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	err := repo.Create(ctx, item1)
	if err != nil {
		t.Fatalf("Failed to create item1: %v", err)
	}

	err = repo.Create(ctx, item2)
	if err != nil {
		t.Fatalf("Failed to create item2: %v", err)
	}

	// Test adding relationship
	err = repo.AddRelationship(ctx, item1.ID, item2.ID, models.RelationshipTypeSupports, 0.8, "test relationship")
	if err != nil {
		t.Errorf("AddRelationship() error = %v", err)
		return
	}

	// Verify relationship was added
	updatedItem1, err := repo.GetByID(ctx, item1.ID)
	if err != nil {
		t.Errorf("Failed to get updated item1: %v", err)
		return
	}

	if len(updatedItem1.Relationships) != 1 {
		t.Errorf("AddRelationship() relationships count = %d, want 1", len(updatedItem1.Relationships))
		return
	}

	rel := updatedItem1.Relationships[0]
	if rel.TargetID != item2.ID {
		t.Errorf("AddRelationship() target ID = %v, want %v", rel.TargetID, item2.ID)
	}

	if rel.Type != models.RelationshipTypeSupports {
		t.Errorf("AddRelationship() type = %v, want %v", rel.Type, models.RelationshipTypeSupports)
	}

	if rel.Strength != 0.8 {
		t.Errorf("AddRelationship() strength = %v, want 0.8", rel.Strength)
	}

	// Test removing relationship
	err = repo.RemoveRelationship(ctx, item1.ID, item2.ID, models.RelationshipTypeSupports)
	if err != nil {
		t.Errorf("RemoveRelationship() error = %v", err)
		return
	}

	// Verify relationship was removed
	updatedItem1Again, err := repo.GetByID(ctx, item1.ID)
	if err != nil {
		t.Errorf("Failed to get updated item1: %v", err)
		return
	}

	if len(updatedItem1Again.Relationships) != 0 {
		t.Errorf("RemoveRelationship() relationships count = %d, want 0", len(updatedItem1Again.Relationships))
	}
}

func TestRepository_GetExpiredItems(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)   // 1 day ago
	futureTime := now.Add(24 * time.Hour)  // 1 day from now

	// Create items with different expiration states
	items := []*models.KnowledgeItem{
		{
			Content:    "Expired item",
			Type:       models.KnowledgeTypeFact,
			Title:      "Expired Item",
			Category:   "Test Category",
			Tags:       []string{"test"},
			Keywords:   []string{"test"},
			Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence: 0.8,
			CreatedBy:  createdBy,
			Validation: models.KnowledgeValidation{
				IsValidated: true,
				ExpiresAt:   &pastTime,
			},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:    "Valid item",
			Type:       models.KnowledgeTypeFact,
			Title:      "Valid Item",
			Category:   "Test Category",
			Tags:       []string{"test"},
			Keywords:   []string{"test"},
			Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence: 0.8,
			CreatedBy:  createdBy,
			Validation: models.KnowledgeValidation{
				IsValidated: true,
				ExpiresAt:   &futureTime,
			},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:    "No expiration item",
			Type:       models.KnowledgeTypeFact,
			Title:      "No Expiration Item",
			Category:   "Test Category",
			Tags:       []string{"test"},
			Keywords:   []string{"test"},
			Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence: 0.8,
			CreatedBy:  createdBy,
			Validation: models.KnowledgeValidation{
				IsValidated: true,
				ExpiresAt:   nil,
			},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	// Create all test items
	for _, item := range items {
		err := repo.Create(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}
	}

	// Test getting expired items
	expiredItems, err := repo.GetExpiredItems(ctx, 10)
	if err != nil {
		t.Errorf("GetExpiredItems() error = %v", err)
		return
	}

	if len(expiredItems) != 1 {
		t.Errorf("GetExpiredItems() count = %d, want 1", len(expiredItems))
		return
	}

	if expiredItems[0].Title != "Expired Item" {
		t.Errorf("GetExpiredItems() returned wrong item = %v, want 'Expired Item'", expiredItems[0].Title)
	}
}

func TestRepository_GetStatistics(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create items with different types and categories
	items := []*models.KnowledgeItem{
		{
			Content:       "Fact 1",
			Type:          models.KnowledgeTypeFact,
			Title:         "Fact 1",
			Category:      "Security",
			Tags:          []string{"test"},
			Keywords:      []string{"test"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence:    0.8,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Fact 2",
			Type:          models.KnowledgeTypeFact,
			Title:         "Fact 2",
			Category:      "Security",
			Tags:          []string{"test"},
			Keywords:      []string{"test"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence:    0.8,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Rule 1",
			Type:          models.KnowledgeTypeRule,
			Title:         "Rule 1",
			Category:      "Policy",
			Tags:          []string{"test"},
			Keywords:      []string{"test"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence:    0.8,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	// Create all test items
	for _, item := range items {
		err := repo.Create(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}
	}

	// Test getting statistics
	stats, err := repo.GetStatistics(ctx)
	if err != nil {
		t.Errorf("GetStatistics() error = %v", err)
		return
	}

	// Check total
	if total, ok := stats["total"].(int64); !ok || total != 3 {
		t.Errorf("GetStatistics() total = %v, want 3", stats["total"])
	}

	// Check by type
	if byType, ok := stats["by_type"].(map[string]int64); ok {
		if byType["fact"] != 2 {
			t.Errorf("GetStatistics() by_type[fact] = %v, want 2", byType["fact"])
		}
		if byType["rule"] != 1 {
			t.Errorf("GetStatistics() by_type[rule] = %v, want 1", byType["rule"])
		}
	} else {
		t.Errorf("GetStatistics() by_type should be map[string]int64")
	}

	// Check by category
	if byCategory, ok := stats["by_category"].(map[string]int64); ok {
		if byCategory["Security"] != 2 {
			t.Errorf("GetStatistics() by_category[Security] = %v, want 2", byCategory["Security"])
		}
		if byCategory["Policy"] != 1 {
			t.Errorf("GetStatistics() by_category[Policy] = %v, want 1", byCategory["Policy"])
		}
	} else {
		t.Errorf("GetStatistics() by_category should be map[string]int64")
	}
}