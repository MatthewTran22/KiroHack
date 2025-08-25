package document

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SupportedFormats defines the supported document formats
var SupportedFormats = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".txt":  true,
}

// ProcessingResult represents the result of document processing
type ProcessingResult struct {
	DocumentID primitive.ObjectID `json:"document_id"`
	Status     string             `json:"status"`
	Message    string             `json:"message"`
}

// ValidationResult represents the result of document validation
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
	Format string   `json:"format"`
	Size   int64    `json:"size"`
}

// Service handles document processing operations
type Service struct {
	db         *mongo.Database
	collection *mongo.Collection
}

// NewService creates a new document processing service
func NewService(db *mongo.Database) *Service {
	return &Service{
		db:         db,
		collection: db.Collection("documents"),
	}
}

// GetDatabase returns the database instance
func (s *Service) GetDatabase() *mongo.Database {
	return s.db
}

// ValidateDocument validates a document file before processing
func (s *Service) ValidateDocument(file *multipart.FileHeader) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
		Size:   file.Size,
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !SupportedFormats[ext] {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported file format: %s", ext))
	} else {
		result.Format = ext
	}

	// Check file size (max 50MB)
	const maxSize = 50 * 1024 * 1024 // 50MB
	if file.Size > maxSize {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", maxSize))
	}

	// Check if file is empty
	if file.Size == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "file is empty")
	}

	// Basic filename validation
	if strings.TrimSpace(file.Filename) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "filename is required")
	}

	return result, nil
}

// UploadDocument handles document upload and initial processing
func (s *Service) UploadDocument(file *multipart.FileHeader, metadata models.DocumentMetadata, uploadedBy primitive.ObjectID) (*ProcessingResult, error) {
	// Validate the document first
	validation, err := s.ValidateDocument(file)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if !validation.Valid {
		return &ProcessingResult{
			Status:  "failed",
			Message: fmt.Sprintf("validation failed: %s", strings.Join(validation.Errors, ", ")),
		}, nil
	}

	// Read file content
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Convert content to string with UTF-8 validation and cleaning
	contentStr := s.sanitizeUTF8Content(content)

	// Create document model
	doc := &models.Document{
		ID:               primitive.NewObjectID(),
		Name:             file.Filename,
		Content:          contentStr,
		ContentType:      file.Header.Get("Content-Type"),
		Size:             file.Size,
		UploadedBy:       uploadedBy,
		UploadedAt:       time.Now(),
		Classification:   models.SecurityClassification{Level: "INTERNAL"}, // Default classification
		Metadata:         metadata,
		ProcessingStatus: models.ProcessingStatusPending,
	}

	// Validate the document model
	if err := doc.Validate(); err != nil {
		return &ProcessingResult{
			Status:  "failed",
			Message: fmt.Sprintf("document validation failed: %s", err.Error()),
		}, nil
	}

	// Insert document into database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = s.collection.InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Start async processing
	go s.processDocumentAsync(doc.ID)

	return &ProcessingResult{
		DocumentID: doc.ID,
		Status:     "uploaded",
		Message:    "Document uploaded successfully and queued for processing",
	}, nil
}

// ProcessDocument processes a document by ID
func (s *Service) ProcessDocument(documentID string) (*models.Document, error) {
	objID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var doc models.Document
	err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	// If already processed, return as is
	if doc.ProcessingStatus == models.ProcessingStatusCompleted {
		return &doc, nil
	}

	// Process the document
	err = s.processDocument(&doc)
	if err != nil {
		// Update status to failed
		s.updateProcessingStatus(objID, models.ProcessingStatusFailed, err.Error())
		return nil, fmt.Errorf("processing failed: %w", err)
	}

	// Update status to completed
	s.updateProcessingStatus(objID, models.ProcessingStatusCompleted, "")

	return &doc, nil
}

// GetProcessingStatus returns the processing status of a document
func (s *Service) GetProcessingStatus(documentID string) (*models.Document, error) {
	objID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doc models.Document
	err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	return &doc, nil
}

// processDocumentAsync processes a document asynchronously
func (s *Service) processDocumentAsync(documentID primitive.ObjectID) {
	// Update status to processing
	s.updateProcessingStatus(documentID, models.ProcessingStatusProcessing, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var doc models.Document
	err := s.collection.FindOne(ctx, bson.M{"_id": documentID}).Decode(&doc)
	if err != nil {
		s.updateProcessingStatus(documentID, models.ProcessingStatusFailed, fmt.Sprintf("failed to find document: %s", err.Error()))
		return
	}

	// Process the document
	err = s.processDocument(&doc)
	if err != nil {
		s.updateProcessingStatus(documentID, models.ProcessingStatusFailed, err.Error())
		return
	}

	// Update status to completed
	s.updateProcessingStatus(documentID, models.ProcessingStatusCompleted, "")
}

// processDocument performs the actual document processing
func (s *Service) processDocument(doc *models.Document) error {
	// Extract text based on file type
	extractedText, err := s.extractText(doc)
	if err != nil {
		return fmt.Errorf("text extraction failed: %w", err)
	}

	// Update document with extracted text
	doc.Content = extractedText

	// Extract metadata
	extractedMetadata, err := s.extractMetadata(doc)
	if err != nil {
		return fmt.Errorf("metadata extraction failed: %w", err)
	}

	// Merge extracted metadata with existing metadata
	s.mergeMetadata(&doc.Metadata, extractedMetadata)

	// Extract entities (basic implementation)
	entities, err := s.extractEntities(extractedText)
	if err != nil {
		return fmt.Errorf("entity extraction failed: %w", err)
	}
	doc.ExtractedEntities = entities

	// Set processing timestamp
	now := time.Now()
	doc.ProcessingTimestamp = &now

	// Update document in database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"content":              doc.Content,
			"metadata":             doc.Metadata,
			"extracted_entities":   doc.ExtractedEntities,
			"processing_timestamp": doc.ProcessingTimestamp,
		},
	}

	_, err = s.collection.UpdateOne(ctx, bson.M{"_id": doc.ID}, update)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// updateProcessingStatus updates the processing status of a document
func (s *Service) updateProcessingStatus(documentID primitive.ObjectID, status models.ProcessingStatus, errorMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	s.collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
}

// sanitizeUTF8Content converts byte content to a valid UTF-8 string
func (s *Service) sanitizeUTF8Content(content []byte) string {
	// First, try to convert directly to string
	str := string(content)
	
	// Check if the string is valid UTF-8
	if utf8.ValidString(str) {
		return str
	}
	
	// If not valid UTF-8, clean it up
	// This will replace invalid UTF-8 sequences with the replacement character
	validStr := strings.ToValidUTF8(str, "�")
	
	// Additional cleaning: remove null bytes and other problematic characters
	validStr = strings.ReplaceAll(validStr, "\x00", "")
	
	// Remove other control characters except common whitespace
	var cleaned strings.Builder
	for _, r := range validStr {
		// Keep printable characters and common whitespace (space, tab, newline, carriage return)
		if r >= 32 || r == '\t' || r == '\n' || r == '\r' {
			cleaned.WriteRune(r)
		}
	}
	
	return cleaned.String()
}
