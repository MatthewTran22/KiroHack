package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentCategory represents the category of a document
type DocumentCategory string

const (
	DocumentCategoryPolicy     DocumentCategory = "policy"
	DocumentCategoryStrategy   DocumentCategory = "strategy"
	DocumentCategoryOperations DocumentCategory = "operations"
	DocumentCategoryTechnology DocumentCategory = "technology"
	DocumentCategoryGeneral    DocumentCategory = "general"
)

// ProcessingStatus represents the current processing status of a document
type ProcessingStatus string

const (
	ProcessingStatusPending    ProcessingStatus = "pending"
	ProcessingStatusProcessing ProcessingStatus = "processing"
	ProcessingStatusCompleted  ProcessingStatus = "completed"
	ProcessingStatusFailed     ProcessingStatus = "failed"
)

// SecurityClassification represents the security classification of a document
type SecurityClassification struct {
	Level                string     `json:"level" bson:"level"` // "PUBLIC", "INTERNAL", "CONFIDENTIAL", "SECRET", "TOP_SECRET"
	Compartments         []string   `json:"compartments" bson:"compartments"`
	Handling             []string   `json:"handling" bson:"handling"`
	DeclassificationDate *time.Time `json:"declassification_date,omitempty" bson:"declassification_date,omitempty"`
}

// DocumentMetadata contains metadata about a document
type DocumentMetadata struct {
	Title        *string                `json:"title,omitempty" bson:"title,omitempty"`
	Author       *string                `json:"author,omitempty" bson:"author,omitempty"`
	Department   *string                `json:"department,omitempty" bson:"department,omitempty"`
	Category     DocumentCategory       `json:"category" bson:"category"`
	Tags         []string               `json:"tags" bson:"tags"`
	Language     string                 `json:"language" bson:"language"`
	CreatedDate  *time.Time             `json:"created_date,omitempty" bson:"created_date,omitempty"`
	LastModified *time.Time             `json:"last_modified,omitempty" bson:"last_modified,omitempty"`
	Version      *string                `json:"version,omitempty" bson:"version,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
}

// Entity represents an extracted entity from a document
type Entity struct {
	Type       string  `json:"type" bson:"type"`
	Value      string  `json:"value" bson:"value"`
	Confidence float64 `json:"confidence" bson:"confidence"`
	StartPos   int     `json:"start_pos" bson:"start_pos"`
	EndPos     int     `json:"end_pos" bson:"end_pos"`
}

// Document represents a document in the system
type Document struct {
	ID                  primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name                string                 `json:"name" bson:"name"`
	Content             string                 `json:"content" bson:"content"`
	ContentType         string                 `json:"content_type" bson:"content_type"`
	Size                int64                  `json:"size" bson:"size"`
	UploadedBy          primitive.ObjectID     `json:"uploaded_by" bson:"uploaded_by"`
	UploadedAt          time.Time              `json:"uploaded_at" bson:"uploaded_at"`
	Classification      SecurityClassification `json:"classification" bson:"classification"`
	Metadata            DocumentMetadata       `json:"metadata" bson:"metadata"`
	ProcessingStatus    ProcessingStatus       `json:"processing_status" bson:"processing_status"`
	Embeddings          []float64              `json:"embeddings,omitempty" bson:"embeddings,omitempty"`
	ExtractedEntities   []Entity               `json:"extracted_entities" bson:"extracted_entities"`
	ProcessingTimestamp *time.Time             `json:"processing_timestamp,omitempty" bson:"processing_timestamp,omitempty"`
	ProcessingError     *string                `json:"processing_error,omitempty" bson:"processing_error,omitempty"`
}

// Validate validates the document model
func (d *Document) Validate() error {
	if d.Name == "" {
		return ErrDocumentNameRequired
	}
	if d.ContentType == "" {
		return ErrDocumentContentTypeRequired
	}
	if d.Size <= 0 {
		return ErrDocumentSizeInvalid
	}
	if d.UploadedBy.IsZero() {
		return ErrDocumentUploadedByRequired
	}
	return nil
}

// IsProcessed returns true if the document has been successfully processed
func (d *Document) IsProcessed() bool {
	return d.ProcessingStatus == ProcessingStatusCompleted
}

// HasEmbeddings returns true if the document has embeddings
func (d *Document) HasEmbeddings() bool {
	return len(d.Embeddings) > 0
}
