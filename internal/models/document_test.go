package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDocument_Validate(t *testing.T) {
	userID := primitive.NewObjectID()

	tests := []struct {
		name    string
		doc     Document
		wantErr error
	}{
		{
			name: "valid document",
			doc: Document{
				Name:        "test.pdf",
				ContentType: "application/pdf",
				Size:        1024,
				UploadedBy:  userID,
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			doc: Document{
				ContentType: "application/pdf",
				Size:        1024,
				UploadedBy:  userID,
			},
			wantErr: ErrDocumentNameRequired,
		},
		{
			name: "missing content type",
			doc: Document{
				Name:       "test.pdf",
				Size:       1024,
				UploadedBy: userID,
			},
			wantErr: ErrDocumentContentTypeRequired,
		},
		{
			name: "invalid size",
			doc: Document{
				Name:        "test.pdf",
				ContentType: "application/pdf",
				Size:        0,
				UploadedBy:  userID,
			},
			wantErr: ErrDocumentSizeInvalid,
		},
		{
			name: "missing uploaded by",
			doc: Document{
				Name:        "test.pdf",
				ContentType: "application/pdf",
				Size:        1024,
			},
			wantErr: ErrDocumentUploadedByRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.doc.Validate()
			if err != tt.wantErr {
				t.Errorf("Document.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDocument_IsProcessed(t *testing.T) {
	tests := []struct {
		name   string
		status ProcessingStatus
		want   bool
	}{
		{
			name:   "completed status",
			status: ProcessingStatusCompleted,
			want:   true,
		},
		{
			name:   "pending status",
			status: ProcessingStatusPending,
			want:   false,
		},
		{
			name:   "processing status",
			status: ProcessingStatusProcessing,
			want:   false,
		},
		{
			name:   "failed status",
			status: ProcessingStatusFailed,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Document{ProcessingStatus: tt.status}
			if got := doc.IsProcessed(); got != tt.want {
				t.Errorf("Document.IsProcessed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocument_HasEmbeddings(t *testing.T) {
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
			doc := Document{Embeddings: tt.embeddings}
			if got := doc.HasEmbeddings(); got != tt.want {
				t.Errorf("Document.HasEmbeddings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocument_BSONSerialization(t *testing.T) {
	userID := primitive.NewObjectID()
	now := time.Now()

	doc := Document{
		ID:          primitive.NewObjectID(),
		Name:        "test-document.pdf",
		Content:     "This is test content",
		ContentType: "application/pdf",
		Size:        1024,
		UploadedBy:  userID,
		UploadedAt:  now,
		Classification: SecurityClassification{
			Level:        "CONFIDENTIAL",
			Compartments: []string{"NOFORN"},
			Handling:     []string{"CONTROLLED"},
		},
		Metadata: DocumentMetadata{
			Title:      stringPtr("Test Document"),
			Author:     stringPtr("Test Author"),
			Department: stringPtr("Test Department"),
			Category:   DocumentCategoryPolicy,
			Tags:       []string{"test", "document"},
			Language:   "en",
		},
		ProcessingStatus: ProcessingStatusCompleted,
		Embeddings:       []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		ExtractedEntities: []Entity{
			{
				Type:       "PERSON",
				Value:      "John Doe",
				Confidence: 0.95,
				StartPos:   10,
				EndPos:     18,
			},
		},
		ProcessingTimestamp: &now,
	}

	// Test BSON marshaling
	data, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document to BSON: %v", err)
	}

	// Test BSON unmarshaling
	var unmarshaled Document
	err = bson.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal document from BSON: %v", err)
	}

	// Verify key fields
	if unmarshaled.Name != doc.Name {
		t.Errorf("Name mismatch: got %v, want %v", unmarshaled.Name, doc.Name)
	}
	if unmarshaled.ContentType != doc.ContentType {
		t.Errorf("ContentType mismatch: got %v, want %v", unmarshaled.ContentType, doc.ContentType)
	}
	if unmarshaled.Size != doc.Size {
		t.Errorf("Size mismatch: got %v, want %v", unmarshaled.Size, doc.Size)
	}
	if unmarshaled.ProcessingStatus != doc.ProcessingStatus {
		t.Errorf("ProcessingStatus mismatch: got %v, want %v", unmarshaled.ProcessingStatus, doc.ProcessingStatus)
	}
	if len(unmarshaled.Embeddings) != len(doc.Embeddings) {
		t.Errorf("Embeddings length mismatch: got %v, want %v", len(unmarshaled.Embeddings), len(doc.Embeddings))
	}
	if len(unmarshaled.ExtractedEntities) != len(doc.ExtractedEntities) {
		t.Errorf("ExtractedEntities length mismatch: got %v, want %v", len(unmarshaled.ExtractedEntities), len(doc.ExtractedEntities))
	}
}

func TestSecurityClassification_BSONSerialization(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	classification := SecurityClassification{
		Level:                "TOP_SECRET",
		Compartments:         []string{"SCI", "TK"},
		Handling:             []string{"NOFORN", "ORCON"},
		DeclassificationDate: &expiry,
	}

	// Test BSON marshaling
	data, err := bson.Marshal(classification)
	if err != nil {
		t.Fatalf("Failed to marshal SecurityClassification to BSON: %v", err)
	}

	// Test BSON unmarshaling
	var unmarshaled SecurityClassification
	err = bson.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SecurityClassification from BSON: %v", err)
	}

	// Verify fields
	if unmarshaled.Level != classification.Level {
		t.Errorf("Level mismatch: got %v, want %v", unmarshaled.Level, classification.Level)
	}
	if len(unmarshaled.Compartments) != len(classification.Compartments) {
		t.Errorf("Compartments length mismatch: got %v, want %v", len(unmarshaled.Compartments), len(classification.Compartments))
	}
	if len(unmarshaled.Handling) != len(classification.Handling) {
		t.Errorf("Handling length mismatch: got %v, want %v", len(unmarshaled.Handling), len(classification.Handling))
	}
	if unmarshaled.DeclassificationDate == nil {
		t.Error("DeclassificationDate should not be nil")
	} else {
		// Compare times with some tolerance for serialization precision differences
		timeDiff := unmarshaled.DeclassificationDate.Sub(*classification.DeclassificationDate)
		if timeDiff < -time.Second || timeDiff > time.Second {
			t.Errorf("DeclassificationDate mismatch: got %v, want %v", unmarshaled.DeclassificationDate, classification.DeclassificationDate)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
