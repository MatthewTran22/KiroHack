package document

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func setupServiceTestDB(t *testing.T) *mongo.Database {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}

	config := &database.Config{
		URI:            "mongodb://admin:password@localhost:27017/test_ai_government_consultant?authSource=admin",
		DatabaseName:   "test_ai_government_consultant_service",
		ConnectTimeout: 5 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    1 * time.Minute,
	}

	mongodb, err := database.NewMongoDB(config)
	if err != nil {
		t.Skipf("Skipping service tests - MongoDB not available: %v", err)
	}

	// Clean up any existing test data
	ctx := context.Background()
	mongodb.Database.Collection("documents").Drop(ctx)

	t.Cleanup(func() {
		mongodb.Database.Drop(ctx)
		mongodb.Close(ctx)
	})

	return mongodb.Database
}

func createTestFileHeader(filename, content string) *multipart.FileHeader {
	if filename == "" {
		// Create a minimal file header for empty filename test
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="file"; filename=""`)
		header.Set("Content-Type", "text/plain")

		return &multipart.FileHeader{
			Filename: "",
			Header:   header,
			Size:     int64(len(content)),
		}
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()

	// Parse the multipart form
	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(1024 * 1024) // 1MB max

	if len(form.File["file"]) == 0 {
		// Fallback for edge cases
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
		header.Set("Content-Type", "text/plain")

		return &multipart.FileHeader{
			Filename: filename,
			Header:   header,
			Size:     int64(len(content)),
		}
	}

	return form.File["file"][0]
}

func createTestFileHeaderWithContentType(filename, content, contentType string) *multipart.FileHeader {
	// Create a more realistic file header
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)

	return &multipart.FileHeader{
		Filename: filename,
		Header:   header,
		Size:     int64(len(content)),
	}
}

func TestService_ValidateDocument(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	tests := []struct {
		name     string
		filename string
		content  string
		wantErr  bool
		errCount int
	}{
		{
			name:     "valid txt file",
			filename: "test.txt",
			content:  "This is a test document",
			wantErr:  false,
			errCount: 0,
		},
		{
			name:     "valid pdf file",
			filename: "test.pdf",
			content:  "PDF content here",
			wantErr:  false,
			errCount: 0,
		},
		{
			name:     "valid docx file",
			filename: "test.docx",
			content:  "DOCX content here",
			wantErr:  false,
			errCount: 0,
		},
		{
			name:     "unsupported file format",
			filename: "test.xyz",
			content:  "Some content",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "empty file",
			filename: "test.txt",
			content:  "",
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "empty filename",
			filename: "",
			content:  "Some content",
			wantErr:  true,
			errCount: 2, // Both unsupported format and filename required errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFileHeader(tt.filename, tt.content)

			result, err := service.ValidateDocument(fileHeader)

			if err != nil {
				t.Errorf("ValidateDocument() error = %v", err)
				return
			}

			if tt.wantErr && result.Valid {
				t.Errorf("ValidateDocument() expected validation to fail, but it passed")
			}

			if !tt.wantErr && !result.Valid {
				t.Errorf("ValidateDocument() expected validation to pass, but it failed: %v", result.Errors)
			}

			if len(result.Errors) != tt.errCount {
				t.Errorf("ValidateDocument() expected %d errors, got %d: %v", tt.errCount, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestService_UploadDocument(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	// Create test user ID
	userID := primitive.NewObjectID()

	tests := []struct {
		name        string
		filename    string
		content     string
		metadata    models.DocumentMetadata
		expectError bool
	}{
		{
			name:     "successful upload",
			filename: "policy.txt",
			content:  "This is a government policy document about regulations and compliance.",
			metadata: models.DocumentMetadata{
				Category: models.DocumentCategoryPolicy,
				Tags:     []string{"policy", "government"},
				Language: "en",
			},
			expectError: false,
		},
		{
			name:     "upload with invalid file",
			filename: "invalid.xyz",
			content:  "Invalid file content",
			metadata: models.DocumentMetadata{
				Category: models.DocumentCategoryGeneral,
			},
			expectError: false, // Should not error, but result should indicate failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFileHeader(tt.filename, tt.content)

			result, err := service.UploadDocument(fileHeader, tt.metadata, userID)

			if tt.expectError && err == nil {
				t.Errorf("UploadDocument() expected error, but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("UploadDocument() unexpected error = %v", err)
			}

			if result != nil {
				if result.Status == "failed" && tt.name == "successful upload" {
					t.Errorf("UploadDocument() expected success, but got failure: %s", result.Message)
				}

				if result.Status == "uploaded" && tt.name == "upload with invalid file" {
					t.Errorf("UploadDocument() expected failure for invalid file, but got success")
				}
			}
		})
	}
}

func TestService_ProcessDocument(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	// Create a test document first
	userID := primitive.NewObjectID()
	fileHeader := createTestFileHeader("test.txt", "This is a test document about government policy and regulations. It contains important information for compliance.")
	metadata := models.DocumentMetadata{
		Category: models.DocumentCategoryPolicy,
		Language: "en",
	}

	result, err := service.UploadDocument(fileHeader, metadata, userID)
	if err != nil {
		t.Fatalf("Failed to upload test document: %v", err)
	}

	if result.Status != "uploaded" {
		t.Fatalf("Expected upload to succeed, got status: %s", result.Status)
	}

	// Wait a moment for async processing to potentially start
	time.Sleep(100 * time.Millisecond)

	// Test processing the document
	doc, err := service.ProcessDocument(result.DocumentID.Hex())
	if err != nil {
		t.Errorf("ProcessDocument() error = %v", err)
		return
	}

	if doc == nil {
		t.Errorf("ProcessDocument() returned nil document")
		return
	}

	// Check that processing was completed
	if doc.ProcessingStatus != models.ProcessingStatusCompleted {
		t.Errorf("ProcessDocument() expected status %s, got %s", models.ProcessingStatusCompleted, doc.ProcessingStatus)
	}

	// Check that content was processed
	if doc.Content == "" {
		t.Errorf("ProcessDocument() document content is empty after processing")
	}

	// Check that metadata was extracted
	if doc.Metadata.Title == nil {
		t.Errorf("ProcessDocument() expected title to be extracted")
	}

	// Check that entities were extracted
	if len(doc.ExtractedEntities) == 0 {
		t.Logf("ProcessDocument() no entities extracted (this may be expected for simple test content)")
	}

	// Check that processing timestamp was set
	if doc.ProcessingTimestamp == nil {
		t.Errorf("ProcessDocument() processing timestamp not set")
	}
}

func TestService_GetProcessingStatus(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	// Create a test document
	userID := primitive.NewObjectID()
	fileHeader := createTestFileHeader("status_test.txt", "Test content for status checking")
	metadata := models.DocumentMetadata{
		Category: models.DocumentCategoryGeneral,
	}

	result, err := service.UploadDocument(fileHeader, metadata, userID)
	if err != nil {
		t.Fatalf("Failed to upload test document: %v", err)
	}

	// Test getting processing status
	doc, err := service.GetProcessingStatus(result.DocumentID.Hex())
	if err != nil {
		t.Errorf("GetProcessingStatus() error = %v", err)
		return
	}

	if doc == nil {
		t.Errorf("GetProcessingStatus() returned nil document")
		return
	}

	// Status should be pending or processing initially
	if doc.ProcessingStatus != models.ProcessingStatusPending && doc.ProcessingStatus != models.ProcessingStatusProcessing {
		t.Errorf("GetProcessingStatus() expected status pending or processing, got %s", doc.ProcessingStatus)
	}

	// Test with invalid document ID
	_, err = service.GetProcessingStatus("invalid-id")
	if err == nil {
		t.Errorf("GetProcessingStatus() expected error for invalid ID, but got none")
	}

	// Test with non-existent document ID
	nonExistentID := primitive.NewObjectID()
	_, err = service.GetProcessingStatus(nonExistentID.Hex())
	if err == nil {
		t.Errorf("GetProcessingStatus() expected error for non-existent document, but got none")
	}
}

func TestService_ExtractText(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	tests := []struct {
		name     string
		filename string
		content  string
		wantErr  bool
	}{
		{
			name:     "extract from txt file",
			filename: "test.txt",
			content:  "This is a plain text document with some content.",
			wantErr:  false,
		},
		{
			name:     "extract from pdf file (simulated)",
			filename: "test.pdf",
			content:  "This is simulated PDF content for testing.",
			wantErr:  false,
		},
		{
			name:     "extract from docx file (simulated)",
			filename: "test.docx",
			content:  "This is simulated DOCX content for testing.",
			wantErr:  false,
		},
		{
			name:     "unsupported format",
			filename: "test.xyz",
			content:  "Unsupported content",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &models.Document{
				Name:    tt.filename,
				Content: tt.content,
			}

			extractedText, err := service.extractText(doc)

			if tt.wantErr && err == nil {
				t.Errorf("extractText() expected error, but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("extractText() unexpected error = %v", err)
			}

			if !tt.wantErr && extractedText == "" {
				t.Errorf("extractText() returned empty text")
			}
		})
	}
}

func TestService_ExtractMetadata(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	tests := []struct {
		name             string
		filename         string
		content          string
		expectedCategory models.DocumentCategory
		expectTags       bool
	}{
		{
			name:             "policy document",
			filename:         "policy.txt",
			content:          "This document outlines government policy and regulations for compliance.",
			expectedCategory: models.DocumentCategoryPolicy,
			expectTags:       true,
		},
		{
			name:             "strategy document",
			filename:         "strategy.txt",
			content:          "Strategic planning document with roadmap and vision for the organization. This includes management analysis and project planning.",
			expectedCategory: models.DocumentCategoryStrategy,
			expectTags:       true,
		},
		{
			name:             "operations document",
			filename:         "operations.txt",
			content:          "Operations manual describing processes and procedures for efficiency. This includes workflow optimization and performance management.",
			expectedCategory: models.DocumentCategoryOperations,
			expectTags:       true,
		},
		{
			name:             "technology document",
			filename:         "tech.txt",
			content:          "Technical documentation for software systems and digital infrastructure. This includes security analysis and project management.",
			expectedCategory: models.DocumentCategoryTechnology,
			expectTags:       true,
		},
		{
			name:             "general document",
			filename:         "general.txt",
			content:          "This document has no special focus or domain.",
			expectedCategory: models.DocumentCategoryGeneral,
			expectTags:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &models.Document{
				Name:    tt.filename,
				Content: tt.content,
			}

			metadata, err := service.extractMetadata(doc)
			if err != nil {
				t.Errorf("extractMetadata() error = %v", err)
				return
			}

			if metadata.Category != tt.expectedCategory {
				t.Errorf("extractMetadata() expected category %s, got %s", tt.expectedCategory, metadata.Category)
			}

			if tt.expectTags && len(metadata.Tags) == 0 {
				t.Errorf("extractMetadata() expected tags to be extracted, but got none")
			}

			if metadata.Title == nil {
				t.Errorf("extractMetadata() expected title to be set")
			}

			if metadata.Language == "" {
				t.Errorf("extractMetadata() expected language to be detected")
			}

			if metadata.CreatedDate == nil {
				t.Errorf("extractMetadata() expected created date to be set")
			}
		})
	}
}

func TestService_ExtractEntities(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	content := `
	Contact information:
	Email: john.doe@government.gov
	Phone: 555-123-4567
	Date: 12/25/2023
	Budget: $1,500,000.00
	Alternative phone: 555.987.6543
	Another email: jane.smith@agency.gov
	`

	entities, err := service.extractEntities(content)
	if err != nil {
		t.Errorf("extractEntities() error = %v", err)
		return
	}

	// Count entities by type
	entityCounts := make(map[string]int)
	for _, entity := range entities {
		entityCounts[entity.Type]++
	}

	// Check that we extracted expected entity types
	expectedTypes := []string{"email", "phone", "date", "money"}
	for _, expectedType := range expectedTypes {
		if entityCounts[expectedType] == 0 {
			t.Errorf("extractEntities() expected to find %s entities, but found none", expectedType)
		}
	}

	// Verify specific extractions
	for _, entity := range entities {
		if entity.Type == "email" {
			if !strings.Contains(entity.Value, "@") {
				t.Errorf("extractEntities() invalid email entity: %s", entity.Value)
			}
		}

		if entity.Type == "money" {
			if !strings.HasPrefix(entity.Value, "$") {
				t.Errorf("extractEntities() invalid money entity: %s", entity.Value)
			}
		}

		// Check confidence scores
		if entity.Confidence <= 0 || entity.Confidence > 1 {
			t.Errorf("extractEntities() invalid confidence score: %f", entity.Confidence)
		}

		// Check positions
		if entity.StartPos < 0 || entity.EndPos <= entity.StartPos {
			t.Errorf("extractEntities() invalid entity positions: start=%d, end=%d", entity.StartPos, entity.EndPos)
		}
	}
}

// Integration test that tests the full workflow
func TestService_FullWorkflow(t *testing.T) {
	db := setupServiceTestDB(t)
	service := NewService(db)

	// Create repository and ensure indexes
	repo := NewRepository(db)
	ctx := context.Background()
	err := repo.CreateIndexes(ctx)
	if err != nil {
		t.Logf("Warning: Could not create indexes: %v", err)
	}

	userID := primitive.NewObjectID()

	// Step 1: Upload document
	fileContent := `
	Government Policy Document
	
	This document outlines the new cybersecurity policy for government agencies.
	
	Contact: security@government.gov
	Phone: 555-123-4567
	Budget allocation: $2,500,000
	Effective date: 01/01/2024
	
	The policy includes regulations for data protection, compliance requirements,
	and security procedures for all government personnel.
	`

	fileHeader := createTestFileHeader("cybersecurity_policy.txt", fileContent)
	metadata := models.DocumentMetadata{
		Department: serviceStringPtr("IT Security"),
		Tags:       []string{"cybersecurity", "policy"},
		Language:   "en",
	}

	result, err := service.UploadDocument(fileHeader, metadata, userID)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if result.Status != "uploaded" {
		t.Fatalf("Expected upload status 'uploaded', got '%s': %s", result.Status, result.Message)
	}

	// Step 2: Check initial status
	doc, err := service.GetProcessingStatus(result.DocumentID.Hex())
	if err != nil {
		t.Fatalf("Failed to get processing status: %v", err)
	}

	if doc.ProcessingStatus != models.ProcessingStatusPending && doc.ProcessingStatus != models.ProcessingStatusProcessing {
		t.Errorf("Expected status pending or processing, got %s", doc.ProcessingStatus)
	}

	// Step 3: Wait a bit for async processing to potentially complete
	time.Sleep(200 * time.Millisecond)

	// Process document synchronously to ensure completion
	processedDoc, err := service.ProcessDocument(result.DocumentID.Hex())
	if err != nil {
		t.Fatalf("Processing failed: %v", err)
	}

	// Step 4: Verify processing results
	if processedDoc.ProcessingStatus != models.ProcessingStatusCompleted {
		t.Errorf("Expected processing status completed, got %s", processedDoc.ProcessingStatus)
	}

	// Check metadata extraction
	if processedDoc.Metadata.Category != models.DocumentCategoryPolicy {
		t.Errorf("Expected category policy, got %s", processedDoc.Metadata.Category)
	}

	if processedDoc.Metadata.Title == nil || *processedDoc.Metadata.Title == "" {
		t.Errorf("Expected title to be extracted")
	}

	// Check entity extraction
	if len(processedDoc.ExtractedEntities) == 0 {
		t.Errorf("Expected entities to be extracted")
	}

	// Verify specific entities
	hasEmail := false
	hasPhone := false
	hasMoney := false
	for _, entity := range processedDoc.ExtractedEntities {
		switch entity.Type {
		case "email":
			hasEmail = true
		case "phone":
			hasPhone = true
		case "money":
			hasMoney = true
		}
	}

	if !hasEmail {
		t.Errorf("Expected email entity to be extracted")
	}
	if !hasPhone {
		t.Errorf("Expected phone entity to be extracted")
	}
	if !hasMoney {
		t.Errorf("Expected money entity to be extracted")
	}

	// Step 5: Test search functionality
	searchFilter := SearchFilter{
		Query:    "cybersecurity",
		Category: models.DocumentCategoryPolicy,
		Limit:    10,
	}

	searchResults, total, err := repo.Search(ctx, searchFilter)
	if err != nil {
		t.Errorf("Search failed: %v", err)
	} else {
		if total == 0 {
			t.Errorf("Expected search to find documents, but got 0 results")
		}

		if len(searchResults) == 0 {
			t.Errorf("Expected search results, but got empty slice")
		}
	}
}

// Helper function to create string pointer for service tests
func serviceStringPtr(s string) *string {
	return &s
}
