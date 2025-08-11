package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KnowledgeType represents the type of knowledge item
type KnowledgeType string

const (
	KnowledgeTypeFact         KnowledgeType = "fact"
	KnowledgeTypeRule         KnowledgeType = "rule"
	KnowledgeTypeProcedure    KnowledgeType = "procedure"
	KnowledgeTypeBestPractice KnowledgeType = "best_practice"
	KnowledgeTypeGuideline    KnowledgeType = "guideline"
	KnowledgeTypeRegulation   KnowledgeType = "regulation"
	KnowledgeTypePrecedent    KnowledgeType = "precedent"
	KnowledgeTypeInsight      KnowledgeType = "insight"
)

// RelationshipType represents the type of relationship between knowledge items
type RelationshipType string

const (
	RelationshipTypeRelatedTo   RelationshipType = "related_to"
	RelationshipTypeSupports    RelationshipType = "supports"
	RelationshipTypeContradicts RelationshipType = "contradicts"
	RelationshipTypeDependsOn   RelationshipType = "depends_on"
	RelationshipTypeSupersedes  RelationshipType = "supersedes"
	RelationshipTypeImplements  RelationshipType = "implements"
	RelationshipTypeExemplifies RelationshipType = "exemplifies"
	RelationshipTypeClarifies   RelationshipType = "clarifies"
)

// KnowledgeRelationship represents a relationship between knowledge items
type KnowledgeRelationship struct {
	Type      RelationshipType   `json:"type" bson:"type"`
	TargetID  primitive.ObjectID `json:"target_id" bson:"target_id"`
	Strength  float64            `json:"strength" bson:"strength"` // 0.0 to 1.0
	Context   string             `json:"context" bson:"context"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// KnowledgeSource represents the source of a knowledge item
type KnowledgeSource struct {
	Type         string             `json:"type" bson:"type"` // "document", "consultation", "manual", "external"
	SourceID     primitive.ObjectID `json:"source_id" bson:"source_id"`
	Reference    string             `json:"reference" bson:"reference"`
	Reliability  float64            `json:"reliability" bson:"reliability"` // 0.0 to 1.0
	LastVerified *time.Time         `json:"last_verified,omitempty" bson:"last_verified,omitempty"`
}

// KnowledgeValidation represents validation information for a knowledge item
type KnowledgeValidation struct {
	IsValidated     bool                `json:"is_validated" bson:"is_validated"`
	ValidatedBy     *primitive.ObjectID `json:"validated_by,omitempty" bson:"validated_by,omitempty"`
	ValidatedAt     *time.Time          `json:"validated_at,omitempty" bson:"validated_at,omitempty"`
	ValidationNotes *string             `json:"validation_notes,omitempty" bson:"validation_notes,omitempty"`
	ExpiresAt       *time.Time          `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
}

// KnowledgeUsage tracks how often and when a knowledge item is used
type KnowledgeUsage struct {
	AccessCount        int64      `json:"access_count" bson:"access_count"`
	LastAccessed       *time.Time `json:"last_accessed,omitempty" bson:"last_accessed,omitempty"`
	UsageContexts      []string   `json:"usage_contexts" bson:"usage_contexts"`
	EffectivenessScore float64    `json:"effectiveness_score" bson:"effectiveness_score"` // 0.0 to 1.0
}

// KnowledgeItem represents a piece of knowledge in the system
type KnowledgeItem struct {
	ID             primitive.ObjectID      `json:"id" bson:"_id,omitempty"`
	Content        string                  `json:"content" bson:"content"`
	Type           KnowledgeType           `json:"type" bson:"type"`
	Title          string                  `json:"title" bson:"title"`
	Summary        *string                 `json:"summary,omitempty" bson:"summary,omitempty"`
	Keywords       []string                `json:"keywords" bson:"keywords"`
	Tags           []string                `json:"tags" bson:"tags"`
	Category       string                  `json:"category" bson:"category"`
	Source         KnowledgeSource         `json:"source" bson:"source"`
	Relationships  []KnowledgeRelationship `json:"relationships" bson:"relationships"`
	Confidence     float64                 `json:"confidence" bson:"confidence"` // 0.0 to 1.0
	Validation     KnowledgeValidation     `json:"validation" bson:"validation"`
	Usage          KnowledgeUsage          `json:"usage" bson:"usage"`
	Embeddings     []float64               `json:"embeddings,omitempty" bson:"embeddings,omitempty"`
	CreatedAt      time.Time               `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at" bson:"updated_at"`
	CreatedBy      primitive.ObjectID      `json:"created_by" bson:"created_by"`
	LastModifiedBy *primitive.ObjectID     `json:"last_modified_by,omitempty" bson:"last_modified_by,omitempty"`
	Version        int                     `json:"version" bson:"version"`
	IsActive       bool                    `json:"is_active" bson:"is_active"`
	Metadata       map[string]interface{}  `json:"metadata" bson:"metadata"`
}

// Validate validates the knowledge item model
func (ki *KnowledgeItem) Validate() error {
	if ki.Content == "" {
		return ErrKnowledgeContentRequired
	}
	if ki.Type == "" {
		return ErrKnowledgeTypeRequired
	}
	if ki.Title == "" {
		return ErrKnowledgeTitleRequired
	}
	if ki.CreatedBy.IsZero() {
		return ErrKnowledgeCreatedByRequired
	}
	if ki.Confidence < 0.0 || ki.Confidence > 1.0 {
		return ErrKnowledgeConfidenceInvalid
	}
	return nil
}

// IsExpired returns true if the knowledge item has expired
func (ki *KnowledgeItem) IsExpired() bool {
	if ki.Validation.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ki.Validation.ExpiresAt)
}

// IsValidated returns true if the knowledge item is validated
func (ki *KnowledgeItem) IsValidated() bool {
	return ki.Validation.IsValidated && !ki.IsExpired()
}

// HasEmbeddings returns true if the knowledge item has embeddings
func (ki *KnowledgeItem) HasEmbeddings() bool {
	return len(ki.Embeddings) > 0
}

// IncrementUsage increments the usage count and updates last accessed time
func (ki *KnowledgeItem) IncrementUsage(context string) {
	ki.Usage.AccessCount++
	now := time.Now()
	ki.Usage.LastAccessed = &now

	// Add context if not already present
	for _, ctx := range ki.Usage.UsageContexts {
		if ctx == context {
			return
		}
	}
	ki.Usage.UsageContexts = append(ki.Usage.UsageContexts, context)
}

// AddRelationship adds a relationship to another knowledge item
func (ki *KnowledgeItem) AddRelationship(relType RelationshipType, targetID primitive.ObjectID, strength float64, context string) {
	relationship := KnowledgeRelationship{
		Type:      relType,
		TargetID:  targetID,
		Strength:  strength,
		Context:   context,
		CreatedAt: time.Now(),
	}
	ki.Relationships = append(ki.Relationships, relationship)
}

// GetRelationshipsByType returns relationships of a specific type
func (ki *KnowledgeItem) GetRelationshipsByType(relType RelationshipType) []KnowledgeRelationship {
	var relationships []KnowledgeRelationship
	for _, rel := range ki.Relationships {
		if rel.Type == relType {
			relationships = append(relationships, rel)
		}
	}
	return relationships
}
