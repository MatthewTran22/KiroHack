package knowledge

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	items         map[primitive.ObjectID]*models.KnowledgeItem
	nextID        primitive.ObjectID
	createIndexes func(ctx context.Context) error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		items:  make(map[primitive.ObjectID]*models.KnowledgeItem),
		nextID: primitive.NewObjectID(),
		createIndexes: func(ctx context.Context) error {
			return nil
		},
	}
}

func (m *MockRepository) CreateIndexes(ctx context.Context) error {
	return m.createIndexes(ctx)
}

func (m *MockRepository) Create(ctx context.Context, item *models.KnowledgeItem) error {
	if item.ID.IsZero() {
		item.ID = m.nextID
		m.nextID = primitive.NewObjectID()
	}
	
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Version = 1
	item.IsActive = true
	
	if err := item.Validate(); err != nil {
		return err
	}
	
	m.items[item.ID] = item
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.KnowledgeItem, error) {
	item, exists := m.items[id]
	if !exists || !item.IsActive {
		return nil, mongo.ErrNoDocuments
	}
	return item, nil
}

func (m *MockRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	item, exists := m.items[id]
	if !exists || !item.IsActive {
		return mongo.ErrNoDocuments
	}
	
	// Apply updates (simplified)
	if content, ok := updates["content"]; ok {
		item.Content = content.(string)
	}
	if title, ok := updates["title"]; ok {
		item.Title = title.(string)
	}
	if confidence, ok := updates["confidence"]; ok {
		item.Confidence = confidence.(float64)
	}
	
	// Handle validation updates
	if isValidated, ok := updates["validation.is_validated"]; ok {
		item.Validation.IsValidated = isValidated.(bool)
	}
	if validatedBy, ok := updates["validation.validated_by"]; ok {
		if validatedByID, ok := validatedBy.(primitive.ObjectID); ok {
			item.Validation.ValidatedBy = &validatedByID
		}
	}
	if validatedAt, ok := updates["validation.validated_at"]; ok {
		if validatedAtTime, ok := validatedAt.(time.Time); ok {
			item.Validation.ValidatedAt = &validatedAtTime
		}
	}
	if validationNotes, ok := updates["validation.validation_notes"]; ok {
		if notes, ok := validationNotes.(string); ok {
			item.Validation.ValidationNotes = &notes
		}
	}
	if expiresAt, ok := updates["validation.expires_at"]; ok {
		if expiresAtTime, ok := expiresAt.(time.Time); ok {
			item.Validation.ExpiresAt = &expiresAtTime
		}
	}
	
	item.UpdatedAt = time.Now()
	item.Version++
	
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	item, exists := m.items[id]
	if !exists || !item.IsActive {
		return mongo.ErrNoDocuments
	}
	
	item.IsActive = false
	item.UpdatedAt = time.Now()
	item.Version++
	
	return nil
}

func (m *MockRepository) Search(ctx context.Context, filter SearchFilter) ([]*models.KnowledgeItem, int64, error) {
	var results []*models.KnowledgeItem
	
	for _, item := range m.items {
		if !item.IsActive {
			continue
		}
		
		// Apply filters (simplified)
		if filter.Type != "" && item.Type != filter.Type {
			continue
		}
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if filter.MinConfidence > 0 && item.Confidence < filter.MinConfidence {
			continue
		}
		
		results = append(results, item)
	}
	
	// Apply limit
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}
	
	return results, int64(len(results)), nil
}

func (m *MockRepository) GetByType(ctx context.Context, knowledgeType models.KnowledgeType, limit int) ([]*models.KnowledgeItem, error) {
	var results []*models.KnowledgeItem
	count := 0
	
	for _, item := range m.items {
		if !item.IsActive || item.Type != knowledgeType {
			continue
		}
		
		results = append(results, item)
		count++
		
		if limit > 0 && count >= limit {
			break
		}
	}
	
	return results, nil
}

func (m *MockRepository) GetBySource(ctx context.Context, sourceType string, sourceID primitive.ObjectID, limit int) ([]*models.KnowledgeItem, error) {
	var results []*models.KnowledgeItem
	count := 0
	
	for _, item := range m.items {
		if !item.IsActive || item.Source.Type != sourceType || item.Source.SourceID != sourceID {
			continue
		}
		
		results = append(results, item)
		count++
		
		if limit > 0 && count >= limit {
			break
		}
	}
	
	return results, nil
}

func (m *MockRepository) GetRelatedItems(ctx context.Context, itemID primitive.ObjectID, relationshipType models.RelationshipType, limit int) ([]*models.KnowledgeItem, error) {
	var results []*models.KnowledgeItem
	count := 0
	
	for _, item := range m.items {
		if !item.IsActive {
			continue
		}
		
		for _, rel := range item.Relationships {
			if rel.TargetID == itemID && (relationshipType == "" || rel.Type == relationshipType) {
				results = append(results, item)
				count++
				break
			}
		}
		
		if limit > 0 && count >= limit {
			break
		}
	}
	
	return results, nil
}

func (m *MockRepository) GetExpiredItems(ctx context.Context, limit int) ([]*models.KnowledgeItem, error) {
	var results []*models.KnowledgeItem
	count := 0
	now := time.Now()
	
	for _, item := range m.items {
		if !item.IsActive {
			continue
		}
		
		if item.Validation.ExpiresAt != nil && item.Validation.ExpiresAt.Before(now) {
			results = append(results, item)
			count++
			
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	
	return results, nil
}

func (m *MockRepository) UpdateUsage(ctx context.Context, id primitive.ObjectID, context string) error {
	item, exists := m.items[id]
	if !exists || !item.IsActive {
		return mongo.ErrNoDocuments
	}
	
	item.Usage.AccessCount++
	now := time.Now()
	item.Usage.LastAccessed = &now
	
	// Add context if not exists
	for _, ctx := range item.Usage.UsageContexts {
		if ctx == context {
			return nil
		}
	}
	item.Usage.UsageContexts = append(item.Usage.UsageContexts, context)
	
	return nil
}

func (m *MockRepository) AddRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType, strength float64, relationshipContext string) error {
	item, exists := m.items[sourceID]
	if !exists || !item.IsActive {
		return mongo.ErrNoDocuments
	}
	
	relationship := models.KnowledgeRelationship{
		Type:      relType,
		TargetID:  targetID,
		Strength:  strength,
		Context:   relationshipContext,
		CreatedAt: time.Now(),
	}
	
	item.Relationships = append(item.Relationships, relationship)
	item.UpdatedAt = time.Now()
	item.Version++
	
	return nil
}

func (m *MockRepository) RemoveRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType) error {
	item, exists := m.items[sourceID]
	if !exists || !item.IsActive {
		return mongo.ErrNoDocuments
	}
	
	var newRelationships []models.KnowledgeRelationship
	for _, rel := range item.Relationships {
		if rel.TargetID != targetID || rel.Type != relType {
			newRelationships = append(newRelationships, rel)
		}
	}
	
	item.Relationships = newRelationships
	item.UpdatedAt = time.Now()
	item.Version++
	
	return nil
}

func (m *MockRepository) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	total := 0
	typeStats := make(map[string]int64)
	categoryStats := make(map[string]int64)
	
	for _, item := range m.items {
		if !item.IsActive {
			continue
		}
		
		total++
		typeStats[string(item.Type)]++
		categoryStats[item.Category]++
	}
	
	stats["total"] = int64(total)
	stats["by_type"] = typeStats
	stats["by_category"] = categoryStats
	
	return stats, nil
}

// Test helper functions
func createTestService() *Service {
	mockRepo := NewMockRepository()
	mockLogger := logger.NewTestLogger()
	
	service := &Service{
		repository: mockRepo,
		logger:     mockLogger,
	}
	
	return service
}

func createTestKnowledgeItem(createdBy primitive.ObjectID) *models.KnowledgeItem {
	return &models.KnowledgeItem{
		Content:    "Test knowledge content",
		Type:       models.KnowledgeTypeFact,
		Title:      "Test Knowledge Item",
		Category:   "Test Category",
		Tags:       []string{"test", "knowledge"},
		Keywords:   []string{"test", "content"},
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
}

// Tests
func TestService_CreateKnowledgeItem(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	tests := []struct {
		name    string
		item    *models.KnowledgeItem
		wantErr bool
	}{
		{
			name:    "valid knowledge item",
			item:    createTestKnowledgeItem(createdBy),
			wantErr: false,
		},
		{
			name: "invalid knowledge item - missing content",
			item: &models.KnowledgeItem{
				Type:      models.KnowledgeTypeFact,
				Title:     "Test",
				CreatedBy: createdBy,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.CreateKnowledgeItem(ctx, tt.item)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateKnowledgeItem() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("CreateKnowledgeItem() error = %v", err)
				return
			}
			
			if result == nil {
				t.Errorf("CreateKnowledgeItem() returned nil result")
				return
			}
			
			if result.Version != 1 {
				t.Errorf("CreateKnowledgeItem() version = %d, want 1", result.Version)
			}
			
			if !result.IsActive {
				t.Errorf("CreateKnowledgeItem() IsActive = false, want true")
			}
		})
	}
}

func TestService_GetKnowledgeItem(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create a test item
	item := createTestKnowledgeItem(createdBy)
	createdItem, err := service.CreateKnowledgeItem(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	// Test getting the item
	retrievedItem, err := service.GetKnowledgeItem(ctx, createdItem.ID)
	if err != nil {
		t.Errorf("GetKnowledgeItem() error = %v", err)
		return
	}

	if retrievedItem.ID != createdItem.ID {
		t.Errorf("GetKnowledgeItem() ID = %v, want %v", retrievedItem.ID, createdItem.ID)
	}

	if retrievedItem.Title != createdItem.Title {
		t.Errorf("GetKnowledgeItem() Title = %v, want %v", retrievedItem.Title, createdItem.Title)
	}

	// Test getting non-existent item
	nonExistentID := primitive.NewObjectID()
	_, err = service.GetKnowledgeItem(ctx, nonExistentID)
	if err == nil {
		t.Errorf("GetKnowledgeItem() expected error for non-existent item, got nil")
	}
}

func TestService_UpdateKnowledgeItem(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create a test item
	item := createTestKnowledgeItem(createdBy)
	createdItem, err := service.CreateKnowledgeItem(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	// Test updating the item
	updates := map[string]interface{}{
		"title":      "Updated Title",
		"content":    "Updated content",
		"confidence": 0.9,
	}

	updatedItem, err := service.UpdateKnowledgeItem(ctx, createdItem.ID, updates)
	if err != nil {
		t.Errorf("UpdateKnowledgeItem() error = %v", err)
		return
	}

	if updatedItem.Title != "Updated Title" {
		t.Errorf("UpdateKnowledgeItem() Title = %v, want 'Updated Title'", updatedItem.Title)
	}

	if updatedItem.Content != "Updated content" {
		t.Errorf("UpdateKnowledgeItem() Content = %v, want 'Updated content'", updatedItem.Content)
	}

	if updatedItem.Confidence != 0.9 {
		t.Errorf("UpdateKnowledgeItem() Confidence = %v, want 0.9", updatedItem.Confidence)
	}

	if updatedItem.Version != 2 {
		t.Errorf("UpdateKnowledgeItem() Version = %v, want 2", updatedItem.Version)
	}
}

func TestService_DeleteKnowledgeItem(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create a test item
	item := createTestKnowledgeItem(createdBy)
	createdItem, err := service.CreateKnowledgeItem(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	// Test deleting the item
	err = service.DeleteKnowledgeItem(ctx, createdItem.ID)
	if err != nil {
		t.Errorf("DeleteKnowledgeItem() error = %v", err)
		return
	}

	// Verify item is soft deleted
	_, err = service.GetKnowledgeItem(ctx, createdItem.ID)
	if err == nil {
		t.Errorf("DeleteKnowledgeItem() item should be soft deleted, but still accessible")
	}
}

func TestService_SearchKnowledge(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items
	items := []*models.KnowledgeItem{
		{
			Content:    "FISMA security requirements",
			Type:       models.KnowledgeTypeRegulation,
			Title:      "FISMA Requirements",
			Category:   "Security",
			Tags:       []string{"security", "regulation"},
			Keywords:   []string{"FISMA", "security"},
			Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence: 0.9,
			CreatedBy:  createdBy,
			Validation: models.KnowledgeValidation{IsValidated: false},
			Usage:      models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:    "Best practices for government IT",
			Type:       models.KnowledgeTypeBestPractice,
			Title:      "IT Best Practices",
			Category:   "Technology",
			Tags:       []string{"technology", "best-practice"},
			Keywords:   []string{"IT", "practices"},
			Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.8},
			Confidence: 0.8,
			CreatedBy:  createdBy,
			Validation: models.KnowledgeValidation{IsValidated: false},
			Usage:      models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	for _, item := range items {
		_, err := service.CreateKnowledgeItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}
	}

	tests := []struct {
		name         string
		filter       SearchFilter
		expectedCount int
	}{
		{
			name:         "search all items",
			filter:       SearchFilter{Limit: 10},
			expectedCount: 2,
		},
		{
			name:         "search by type",
			filter:       SearchFilter{Type: models.KnowledgeTypeRegulation, Limit: 10},
			expectedCount: 1,
		},
		{
			name:         "search by category",
			filter:       SearchFilter{Category: "Security", Limit: 10},
			expectedCount: 1,
		},
		{
			name:         "search by confidence",
			filter:       SearchFilter{MinConfidence: 0.85, Limit: 10},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := service.SearchKnowledge(ctx, tt.filter)
			if err != nil {
				t.Errorf("SearchKnowledge() error = %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("SearchKnowledge() results count = %d, want %d", len(results), tt.expectedCount)
			}

			if total != int64(tt.expectedCount) {
				t.Errorf("SearchKnowledge() total = %d, want %d", total, tt.expectedCount)
			}
		})
	}
}

func TestService_AddRelationship(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items
	item1 := createTestKnowledgeItem(createdBy)
	item1.Title = "Item 1"
	createdItem1, err := service.CreateKnowledgeItem(ctx, item1)
	if err != nil {
		t.Fatalf("Failed to create test item 1: %v", err)
	}

	item2 := createTestKnowledgeItem(createdBy)
	item2.Title = "Item 2"
	createdItem2, err := service.CreateKnowledgeItem(ctx, item2)
	if err != nil {
		t.Fatalf("Failed to create test item 2: %v", err)
	}

	// Test adding relationship
	err = service.AddRelationship(ctx, createdItem1.ID, createdItem2.ID, models.RelationshipTypeRelatedTo, 0.8, "test relationship")
	if err != nil {
		t.Errorf("AddRelationship() error = %v", err)
		return
	}

	// Verify relationship was added
	updatedItem1, err := service.GetKnowledgeItem(ctx, createdItem1.ID)
	if err != nil {
		t.Errorf("Failed to get updated item: %v", err)
		return
	}

	if len(updatedItem1.Relationships) != 1 {
		t.Errorf("AddRelationship() relationships count = %d, want 1", len(updatedItem1.Relationships))
		return
	}

	rel := updatedItem1.Relationships[0]
	if rel.TargetID != createdItem2.ID {
		t.Errorf("AddRelationship() target ID = %v, want %v", rel.TargetID, createdItem2.ID)
	}

	if rel.Type != models.RelationshipTypeRelatedTo {
		t.Errorf("AddRelationship() type = %v, want %v", rel.Type, models.RelationshipTypeRelatedTo)
	}

	if rel.Strength != 0.8 {
		t.Errorf("AddRelationship() strength = %v, want 0.8", rel.Strength)
	}
}

func TestService_ValidateKnowledgeItem(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()
	validatedBy := primitive.NewObjectID()

	// Create a test item
	item := createTestKnowledgeItem(createdBy)
	createdItem, err := service.CreateKnowledgeItem(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	// Test validating the item
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days from now
	err = service.ValidateKnowledgeItem(ctx, createdItem.ID, validatedBy, "Validated for testing", &expiresAt)
	if err != nil {
		t.Errorf("ValidateKnowledgeItem() error = %v", err)
		return
	}

	// Verify validation was applied
	validatedItem, err := service.GetKnowledgeItem(ctx, createdItem.ID)
	if err != nil {
		t.Errorf("Failed to get validated item: %v", err)
		return
	}

	if !validatedItem.Validation.IsValidated {
		t.Errorf("ValidateKnowledgeItem() IsValidated = false, want true")
	}

	if validatedItem.Validation.ValidatedBy == nil || *validatedItem.Validation.ValidatedBy != validatedBy {
		t.Errorf("ValidateKnowledgeItem() ValidatedBy = %v, want %v", validatedItem.Validation.ValidatedBy, validatedBy)
	}

	if validatedItem.Validation.ExpiresAt == nil {
		t.Errorf("ValidateKnowledgeItem() ExpiresAt should not be nil")
	}
}

func TestService_BuildKnowledgeGraph(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items with relationships
	item1 := createTestKnowledgeItem(createdBy)
	item1.Title = "Security Policy"
	item1.Type = models.KnowledgeTypeRule
	createdItem1, err := service.CreateKnowledgeItem(ctx, item1)
	if err != nil {
		t.Fatalf("Failed to create test item 1: %v", err)
	}

	item2 := createTestKnowledgeItem(createdBy)
	item2.Title = "Security Procedure"
	item2.Type = models.KnowledgeTypeProcedure
	createdItem2, err := service.CreateKnowledgeItem(ctx, item2)
	if err != nil {
		t.Fatalf("Failed to create test item 2: %v", err)
	}

	// Add relationship
	err = service.AddRelationship(ctx, createdItem1.ID, createdItem2.ID, models.RelationshipTypeImplements, 0.9, "policy implementation")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Test building knowledge graph
	filter := SearchFilter{Limit: 10}
	graph, err := service.GetKnowledgeGraph(ctx, filter)
	if err != nil {
		t.Errorf("GetKnowledgeGraph() error = %v", err)
		return
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("GetKnowledgeGraph() nodes count = %d, want 2", len(graph.Nodes))
	}

	if len(graph.Edges) != 1 {
		t.Errorf("GetKnowledgeGraph() edges count = %d, want 1", len(graph.Edges))
	}

	// Verify edge properties
	if len(graph.Edges) > 0 {
		edge := graph.Edges[0]
		if edge.Source != createdItem1.ID.Hex() {
			t.Errorf("GetKnowledgeGraph() edge source = %v, want %v", edge.Source, createdItem1.ID.Hex())
		}
		if edge.Target != createdItem2.ID.Hex() {
			t.Errorf("GetKnowledgeGraph() edge target = %v, want %v", edge.Target, createdItem2.ID.Hex())
		}
		if edge.Type != models.RelationshipTypeImplements {
			t.Errorf("GetKnowledgeGraph() edge type = %v, want %v", edge.Type, models.RelationshipTypeImplements)
		}
	}
}

func TestService_ValidateConsistency(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items with potential consistency issues
	expiredItem := createTestKnowledgeItem(createdBy)
	expiredItem.Title = "Expired Knowledge"
	pastTime := time.Now().Add(-24 * time.Hour) // 1 day ago
	expiredItem.Validation = models.KnowledgeValidation{
		IsValidated: true,
		ExpiresAt:   &pastTime,
	}
	_, err := service.CreateKnowledgeItem(ctx, expiredItem)
	if err != nil {
		t.Fatalf("Failed to create expired item: %v", err)
	}

	// Create high usage, low confidence item
	highUsageItem := createTestKnowledgeItem(createdBy)
	highUsageItem.Title = "High Usage Low Confidence"
	highUsageItem.Confidence = 0.4
	highUsageItem.Usage.AccessCount = 15
	createdHighUsageItem, err := service.CreateKnowledgeItem(ctx, highUsageItem)
	if err != nil {
		t.Fatalf("Failed to create high usage item: %v", err)
	}
	
	// Update the usage count after creation since CreateKnowledgeItem resets it
	err = service.GetRepository().UpdateUsage(ctx, createdHighUsageItem.ID, "test")
	for i := 0; i < 14; i++ { // Add 14 more to get to 15 total
		err = service.GetRepository().UpdateUsage(ctx, createdHighUsageItem.ID, "test")
		if err != nil {
			t.Fatalf("Failed to update usage: %v", err)
		}
	}

	// Test consistency validation
	issues, err := service.ValidateConsistency(ctx)
	if err != nil {
		t.Errorf("ValidateConsistency() error = %v", err)
		return
	}

	if len(issues) < 2 {
		t.Errorf("ValidateConsistency() issues count = %d, want at least 2", len(issues))
	}

	// Check for expired issue
	foundExpired := false
	foundHighUsage := false
	for _, issue := range issues {
		if issue.Type == "expired" {
			foundExpired = true
		}
		if issue.Type == "low_confidence_high_usage" {
			foundHighUsage = true
		}
	}

	if !foundExpired {
		t.Errorf("ValidateConsistency() should find expired issue")
	}

	if !foundHighUsage {
		t.Errorf("ValidateConsistency() should find high usage low confidence issue")
	}
}

func TestService_ExtractKnowledgeFromDocument(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	extractedBy := primitive.NewObjectID()

	// Create test document
	document := &models.Document{
		ID:      primitive.NewObjectID(),
		Name:    "Test Policy Document",
		Content: "Government agencies must implement security controls. The procedure requires following these steps: first, assess risks; second, implement controls. It is recommended to use best practices for security management.",
		Metadata: models.DocumentMetadata{
			Category: models.DocumentCategoryPolicy,
			Tags:     []string{"security", "policy"},
		},
	}

	// Test knowledge extraction
	extractedItems, err := service.ExtractKnowledgeFromDocument(ctx, document, extractedBy)
	if err != nil {
		t.Errorf("ExtractKnowledgeFromDocument() error = %v", err)
		return
	}

	if len(extractedItems) == 0 {
		t.Errorf("ExtractKnowledgeFromDocument() should extract at least one item")
		return
	}

	// Verify extracted items have correct properties
	for _, item := range extractedItems {
		if item.Source.Type != "document" {
			t.Errorf("ExtractKnowledgeFromDocument() source type = %v, want 'document'", item.Source.Type)
		}
		if item.Source.SourceID != document.ID {
			t.Errorf("ExtractKnowledgeFromDocument() source ID = %v, want %v", item.Source.SourceID, document.ID)
		}
		if item.CreatedBy != extractedBy {
			t.Errorf("ExtractKnowledgeFromDocument() created by = %v, want %v", item.CreatedBy, extractedBy)
		}
		if item.Category != string(document.Metadata.Category) {
			t.Errorf("ExtractKnowledgeFromDocument() category = %v, want %v", item.Category, string(document.Metadata.Category))
		}
	}

	// Check for different types of extracted knowledge
	foundTypes := make(map[models.KnowledgeType]bool)
	for _, item := range extractedItems {
		foundTypes[item.Type] = true
	}

	// Should extract at least facts and rules from the test content
	if !foundTypes[models.KnowledgeTypeFact] && !foundTypes[models.KnowledgeTypeRule] {
		t.Errorf("ExtractKnowledgeFromDocument() should extract facts or rules")
	}
}

func TestService_ExportImportKnowledge(t *testing.T) {
	service := createTestService()
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items
	item1 := createTestKnowledgeItem(createdBy)
	item1.Title = "Export Test Item 1"
	_, err := service.CreateKnowledgeItem(ctx, item1)
	if err != nil {
		t.Fatalf("Failed to create test item 1: %v", err)
	}

	item2 := createTestKnowledgeItem(createdBy)
	item2.Title = "Export Test Item 2"
	_, err = service.CreateKnowledgeItem(ctx, item2)
	if err != nil {
		t.Fatalf("Failed to create test item 2: %v", err)
	}

	// Test export
	exportOptions := KnowledgeExportOptions{
		Format:               ExportFormatJSON,
		IncludeRelationships: true,
		IncludeMetadata:      true,
		IncludeUsageStats:    true,
		MinConfidence:        0.0,
		ValidatedOnly:        false,
	}

	exportData, err := service.ExportKnowledge(ctx, exportOptions)
	if err != nil {
		t.Errorf("ExportKnowledge() error = %v", err)
		return
	}

	if len(exportData) == 0 {
		t.Errorf("ExportKnowledge() should return non-empty data")
		return
	}

	// Test import
	importResult, err := service.ImportKnowledge(ctx, exportData, ExportFormatJSON, createdBy)
	if err != nil {
		t.Errorf("ImportKnowledge() error = %v", err)
		return
	}

	if importResult.TotalItems == 0 {
		t.Errorf("ImportKnowledge() should import at least one item")
	}

	if importResult.ImportedItems == 0 {
		t.Errorf("ImportKnowledge() should successfully import at least one item")
	}

	if importResult.ErrorItems > 0 {
		t.Errorf("ImportKnowledge() should not have errors, got %d errors: %v", importResult.ErrorItems, importResult.Errors)
	}
}