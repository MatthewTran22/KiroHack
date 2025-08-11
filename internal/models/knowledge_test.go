package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestKnowledgeItem_Validate(t *testing.T) {
	createdBy := primitive.NewObjectID()

	tests := []struct {
		name    string
		item    KnowledgeItem
		wantErr error
	}{
		{
			name: "valid knowledge item",
			item: KnowledgeItem{
				Content:    "FISMA requires federal agencies to implement security controls",
				Type:       KnowledgeTypeRegulation,
				Title:      "FISMA Security Requirements",
				CreatedBy:  createdBy,
				Confidence: 0.95,
			},
			wantErr: nil,
		},
		{
			name: "missing content",
			item: KnowledgeItem{
				Type:       KnowledgeTypeRegulation,
				Title:      "FISMA Security Requirements",
				CreatedBy:  createdBy,
				Confidence: 0.95,
			},
			wantErr: ErrKnowledgeContentRequired,
		},
		{
			name: "missing type",
			item: KnowledgeItem{
				Content:    "FISMA requires federal agencies to implement security controls",
				Title:      "FISMA Security Requirements",
				CreatedBy:  createdBy,
				Confidence: 0.95,
			},
			wantErr: ErrKnowledgeTypeRequired,
		},
		{
			name: "missing title",
			item: KnowledgeItem{
				Content:    "FISMA requires federal agencies to implement security controls",
				Type:       KnowledgeTypeRegulation,
				CreatedBy:  createdBy,
				Confidence: 0.95,
			},
			wantErr: ErrKnowledgeTitleRequired,
		},
		{
			name: "missing created by",
			item: KnowledgeItem{
				Content:    "FISMA requires federal agencies to implement security controls",
				Type:       KnowledgeTypeRegulation,
				Title:      "FISMA Security Requirements",
				Confidence: 0.95,
			},
			wantErr: ErrKnowledgeCreatedByRequired,
		},
		{
			name: "invalid confidence - too low",
			item: KnowledgeItem{
				Content:    "FISMA requires federal agencies to implement security controls",
				Type:       KnowledgeTypeRegulation,
				Title:      "FISMA Security Requirements",
				CreatedBy:  createdBy,
				Confidence: -0.1,
			},
			wantErr: ErrKnowledgeConfidenceInvalid,
		},
		{
			name: "invalid confidence - too high",
			item: KnowledgeItem{
				Content:    "FISMA requires federal agencies to implement security controls",
				Type:       KnowledgeTypeRegulation,
				Title:      "FISMA Security Requirements",
				CreatedBy:  createdBy,
				Confidence: 1.1,
			},
			wantErr: ErrKnowledgeConfidenceInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Validate()
			if err != tt.wantErr {
				t.Errorf("KnowledgeItem.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKnowledgeItem_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "not expired - future date",
			expiresAt: &future,
			want:      false,
		},
		{
			name:      "expired - past date",
			expiresAt: &past,
			want:      true,
		},
		{
			name:      "no expiration date",
			expiresAt: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := KnowledgeItem{
				Validation: KnowledgeValidation{
					ExpiresAt: tt.expiresAt,
				},
			}
			if got := item.IsExpired(); got != tt.want {
				t.Errorf("KnowledgeItem.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKnowledgeItem_IsValidated(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name        string
		isValidated bool
		expiresAt   *time.Time
		want        bool
	}{
		{
			name:        "validated and not expired",
			isValidated: true,
			expiresAt:   &future,
			want:        true,
		},
		{
			name:        "validated but expired",
			isValidated: true,
			expiresAt:   &past,
			want:        false,
		},
		{
			name:        "not validated",
			isValidated: false,
			expiresAt:   &future,
			want:        false,
		},
		{
			name:        "validated with no expiration",
			isValidated: true,
			expiresAt:   nil,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := KnowledgeItem{
				Validation: KnowledgeValidation{
					IsValidated: tt.isValidated,
					ExpiresAt:   tt.expiresAt,
				},
			}
			if got := item.IsValidated(); got != tt.want {
				t.Errorf("KnowledgeItem.IsValidated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKnowledgeItem_HasEmbeddings(t *testing.T) {
	tests := []struct {
		name       string
		embeddings []float64
		want       bool
	}{
		{
			name:       "has embeddings",
			embeddings: []float64{0.1, 0.2, 0.3},
			want:       true,
		},
		{
			name:       "no embeddings",
			embeddings: []float64{},
			want:       false,
		},
		{
			name:       "nil embeddings",
			embeddings: nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := KnowledgeItem{Embeddings: tt.embeddings}
			if got := item.HasEmbeddings(); got != tt.want {
				t.Errorf("KnowledgeItem.HasEmbeddings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKnowledgeItem_IncrementUsage(t *testing.T) {
	item := KnowledgeItem{
		Usage: KnowledgeUsage{
			AccessCount:   5,
			UsageContexts: []string{"policy", "compliance"},
		},
	}

	beforeUpdate := time.Now()
	item.IncrementUsage("security")
	afterUpdate := time.Now()

	// Check access count incremented
	if item.Usage.AccessCount != 6 {
		t.Errorf("AccessCount should be 6, got %d", item.Usage.AccessCount)
	}

	// Check last accessed time updated
	if item.Usage.LastAccessed == nil {
		t.Error("LastAccessed should not be nil")
	} else if item.Usage.LastAccessed.Before(beforeUpdate) || item.Usage.LastAccessed.After(afterUpdate) {
		t.Errorf("LastAccessed should be between %v and %v, got %v",
			beforeUpdate, afterUpdate, item.Usage.LastAccessed)
	}

	// Check new context added
	found := false
	for _, ctx := range item.Usage.UsageContexts {
		if ctx == "security" {
			found = true
			break
		}
	}
	if !found {
		t.Error("New context 'security' should be added to UsageContexts")
	}

	// Test adding duplicate context
	item.IncrementUsage("security")
	if item.Usage.AccessCount != 7 {
		t.Errorf("AccessCount should be 7, got %d", item.Usage.AccessCount)
	}

	// Should not add duplicate context
	securityCount := 0
	for _, ctx := range item.Usage.UsageContexts {
		if ctx == "security" {
			securityCount++
		}
	}
	if securityCount != 1 {
		t.Errorf("Should have only one 'security' context, got %d", securityCount)
	}
}

func TestKnowledgeItem_AddRelationship(t *testing.T) {
	targetID := primitive.NewObjectID()
	item := KnowledgeItem{
		Relationships: []KnowledgeRelationship{},
	}

	beforeAdd := time.Now()
	item.AddRelationship(RelationshipTypeRelatedTo, targetID, 0.8, "test context")
	afterAdd := time.Now()

	if len(item.Relationships) != 1 {
		t.Errorf("Should have 1 relationship, got %d", len(item.Relationships))
	}

	rel := item.Relationships[0]
	if rel.Type != RelationshipTypeRelatedTo {
		t.Errorf("Relationship type should be %v, got %v", RelationshipTypeRelatedTo, rel.Type)
	}
	if rel.TargetID != targetID {
		t.Errorf("Target ID should be %v, got %v", targetID, rel.TargetID)
	}
	if rel.Strength != 0.8 {
		t.Errorf("Strength should be 0.8, got %f", rel.Strength)
	}
	if rel.Context != "test context" {
		t.Errorf("Context should be 'test context', got %v", rel.Context)
	}
	if rel.CreatedAt.Before(beforeAdd) || rel.CreatedAt.After(afterAdd) {
		t.Errorf("CreatedAt should be between %v and %v, got %v",
			beforeAdd, afterAdd, rel.CreatedAt)
	}
}

func TestKnowledgeItem_GetRelationshipsByType(t *testing.T) {
	targetID1 := primitive.NewObjectID()
	targetID2 := primitive.NewObjectID()
	targetID3 := primitive.NewObjectID()

	item := KnowledgeItem{
		Relationships: []KnowledgeRelationship{
			{Type: RelationshipTypeRelatedTo, TargetID: targetID1},
			{Type: RelationshipTypeSupports, TargetID: targetID2},
			{Type: RelationshipTypeRelatedTo, TargetID: targetID3},
		},
	}

	// Test getting related_to relationships
	relatedTo := item.GetRelationshipsByType(RelationshipTypeRelatedTo)
	if len(relatedTo) != 2 {
		t.Errorf("Should have 2 related_to relationships, got %d", len(relatedTo))
	}

	// Test getting supports relationships
	supports := item.GetRelationshipsByType(RelationshipTypeSupports)
	if len(supports) != 1 {
		t.Errorf("Should have 1 supports relationship, got %d", len(supports))
	}

	// Test getting non-existent relationship type
	contradicts := item.GetRelationshipsByType(RelationshipTypeContradicts)
	if len(contradicts) != 0 {
		t.Errorf("Should have 0 contradicts relationships, got %d", len(contradicts))
	}
}

func TestKnowledgeItem_BSONSerialization(t *testing.T) {
	createdBy := primitive.NewObjectID()
	validatedBy := primitive.NewObjectID()
	sourceID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	now := time.Now()
	expiry := now.Add(24 * time.Hour)

	item := KnowledgeItem{
		ID:       primitive.NewObjectID(),
		Content:  "FISMA requires federal agencies to implement comprehensive security controls",
		Type:     KnowledgeTypeRegulation,
		Title:    "FISMA Security Control Requirements",
		Summary:  stringPtr("Federal agencies must implement FISMA security controls"),
		Keywords: []string{"FISMA", "security", "controls", "federal"},
		Tags:     []string{"security", "regulation", "compliance"},
		Category: "Information Security",
		Source: KnowledgeSource{
			Type:         "document",
			SourceID:     sourceID,
			Reference:    "FISMA Act Section 3544",
			Reliability:  1.0,
			LastVerified: &now,
		},
		Relationships: []KnowledgeRelationship{
			{
				Type:      RelationshipTypeRelatedTo,
				TargetID:  targetID,
				Strength:  0.9,
				Context:   "Both relate to federal security requirements",
				CreatedAt: now,
			},
		},
		Confidence: 0.95,
		Validation: KnowledgeValidation{
			IsValidated:     true,
			ValidatedBy:     &validatedBy,
			ValidatedAt:     &now,
			ValidationNotes: stringPtr("Verified against official FISMA documentation"),
			ExpiresAt:       &expiry,
		},
		Usage: KnowledgeUsage{
			AccessCount:        10,
			LastAccessed:       &now,
			UsageContexts:      []string{"policy", "compliance", "security"},
			EffectivenessScore: 0.85,
		},
		Embeddings:     []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      createdBy,
		LastModifiedBy: &validatedBy,
		Version:        2,
		IsActive:       true,
		Metadata: map[string]interface{}{
			"priority":   "high",
			"department": "IT Security",
			"reviewed":   true,
		},
	}

	// Test BSON marshaling
	data, err := bson.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal knowledge item to BSON: %v", err)
	}

	// Test BSON unmarshaling
	var unmarshaled KnowledgeItem
	err = bson.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal knowledge item from BSON: %v", err)
	}

	// Verify key fields
	if unmarshaled.Content != item.Content {
		t.Errorf("Content mismatch: got %v, want %v", unmarshaled.Content, item.Content)
	}
	if unmarshaled.Type != item.Type {
		t.Errorf("Type mismatch: got %v, want %v", unmarshaled.Type, item.Type)
	}
	if unmarshaled.Title != item.Title {
		t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, item.Title)
	}
	if unmarshaled.Confidence != item.Confidence {
		t.Errorf("Confidence mismatch: got %v, want %v", unmarshaled.Confidence, item.Confidence)
	}
	if unmarshaled.Version != item.Version {
		t.Errorf("Version mismatch: got %v, want %v", unmarshaled.Version, item.Version)
	}
	if unmarshaled.IsActive != item.IsActive {
		t.Errorf("IsActive mismatch: got %v, want %v", unmarshaled.IsActive, item.IsActive)
	}

	// Verify arrays
	if len(unmarshaled.Keywords) != len(item.Keywords) {
		t.Errorf("Keywords length mismatch: got %v, want %v", len(unmarshaled.Keywords), len(item.Keywords))
	}
	if len(unmarshaled.Tags) != len(item.Tags) {
		t.Errorf("Tags length mismatch: got %v, want %v", len(unmarshaled.Tags), len(item.Tags))
	}
	if len(unmarshaled.Embeddings) != len(item.Embeddings) {
		t.Errorf("Embeddings length mismatch: got %v, want %v", len(unmarshaled.Embeddings), len(item.Embeddings))
	}
	if len(unmarshaled.Relationships) != len(item.Relationships) {
		t.Errorf("Relationships length mismatch: got %v, want %v", len(unmarshaled.Relationships), len(item.Relationships))
	}

	// Verify nested structures
	if unmarshaled.Source.Type != item.Source.Type {
		t.Errorf("Source.Type mismatch: got %v, want %v", unmarshaled.Source.Type, item.Source.Type)
	}
	if unmarshaled.Source.Reliability != item.Source.Reliability {
		t.Errorf("Source.Reliability mismatch: got %v, want %v", unmarshaled.Source.Reliability, item.Source.Reliability)
	}

	if unmarshaled.Validation.IsValidated != item.Validation.IsValidated {
		t.Errorf("Validation.IsValidated mismatch: got %v, want %v",
			unmarshaled.Validation.IsValidated, item.Validation.IsValidated)
	}

	if unmarshaled.Usage.AccessCount != item.Usage.AccessCount {
		t.Errorf("Usage.AccessCount mismatch: got %v, want %v",
			unmarshaled.Usage.AccessCount, item.Usage.AccessCount)
	}
	if unmarshaled.Usage.EffectivenessScore != item.Usage.EffectivenessScore {
		t.Errorf("Usage.EffectivenessScore mismatch: got %v, want %v",
			unmarshaled.Usage.EffectivenessScore, item.Usage.EffectivenessScore)
	}

	// Verify metadata
	if len(unmarshaled.Metadata) != len(item.Metadata) {
		t.Errorf("Metadata length mismatch: got %v, want %v", len(unmarshaled.Metadata), len(item.Metadata))
	}
}
