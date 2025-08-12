package document

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *mongo.Database {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}

	config := &database.Config{
		URI:            "mongodb://admin:password@localhost:27017/test_ai_government_consultant?authSource=admin",
		DatabaseName:   "test_ai_government_consultant_repo",
		ConnectTimeout: 5 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    1 * time.Minute,
	}

	mongodb, err := database.NewMongoDB(config)
	if err != nil {
		t.Skipf("Skipping repository tests - MongoDB not available: %v", err)
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

// createTestDocument creates a test document for testing
func createTestDocument(userID primitive.ObjectID) *models.Document {
	return &models.Document{
		ID:          primitive.NewObjectID(),
		Name:        "test-document.pdf",
		Content:     "This is test document content for testing purposes.",
		ContentType: "application/pdf",
		Size:        1024,
		UploadedBy:  userID,
		UploadedAt:  time.Now(),
		Classification: models.SecurityClassification{
			Level:        "PUBLIC",
			Compartments: []string{},
			Handling:     []string{},
		},
		Metadata: models.DocumentMetadata{
			Title:        stringPtr("Test Document"),
			Author:       stringPtr("Test Author"),
			Department:   stringPtr("Test Department"),
			Category:     models.DocumentCategoryGeneral,
			Tags:         []string{"test", "document"},
			Language:     "en",
			CreatedDate:  timePtr(time.Now()),
			CustomFields: make(map[string]interface{}),
		},
		ProcessingStatus:  models.ProcessingStatusPending,
		ExtractedEntities: []models.Entity{},
	}
}

func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestRepository_CreateIndexes(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	ctx := context.Background()

	err := repo.CreateIndexes(ctx)
	if err != nil {
		t.Fatalf("CreateIndexes failed: %v", err)
	}

	// Verify indexes were created by checking collection indexes
	indexes := repo.collection.Indexes()
	cursor, err := indexes.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list indexes: %v", err)
	}
	defer cursor.Close(ctx)

	indexCount := 0
	for cursor.Next(ctx) {
		indexCount++
	}

	// Should have at least the default _id index plus our custom indexes
	if indexCount < 7 {
		t.Errorf("Expected at least 7 indexes, got %d", indexCount)
	}
}

func TestRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	doc := createTestDocument(userID)

	err := repo.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify document was created
	if doc.ID.IsZero() {
		t.Error("Document ID should be set after creation")
	}

	// Verify document exists in database
	var result models.Document
	err = repo.collection.FindOne(ctx, bson.M{"_id": doc.ID}).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to find created document: %v", err)
	}

	if result.Name != doc.Name {
		t.Errorf("Expected name %s, got %s", doc.Name, result.Name)
	}
}

func TestRepository_Create_WithExistingID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	doc := createTestDocument(userID)
	existingID := primitive.NewObjectID()
	doc.ID = existingID

	err := repo.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify the existing ID was preserved
	if doc.ID != existingID {
		t.Errorf("Expected ID %s to be preserved, got %s", existingID.Hex(), doc.ID.Hex())
	}
}

func TestRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	doc := createTestDocument(userID)

	// Create document first
	err := repo.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get document by ID
	result, err := repo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if result.ID != doc.ID {
		t.Errorf("Expected ID %s, got %s", doc.ID.Hex(), result.ID.Hex())
	}
	if result.Name != doc.Name {
		t.Errorf("Expected name %s, got %s", doc.Name, result.Name)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	nonExistentID := primitive.NewObjectID()

	_, err := repo.GetByID(ctx, nonExistentID)
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
	if err.Error() != "document not found" {
		t.Errorf("Expected 'document not found' error, got: %v", err)
	}
}

func TestRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	doc := createTestDocument(userID)

	// Create document first
	err := repo.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update document
	update := bson.M{
		"$set": bson.M{
			"processing_status": models.ProcessingStatusCompleted,
			"content":           "Updated content",
		},
	}

	err = repo.Update(ctx, doc.ID, update)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	result, err := repo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if result.ProcessingStatus != models.ProcessingStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ProcessingStatusCompleted, result.ProcessingStatus)
	}
	if result.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got %s", result.Content)
	}
}

func TestRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	nonExistentID := primitive.NewObjectID()
	update := bson.M{"$set": bson.M{"content": "test"}}

	err := repo.Update(ctx, nonExistentID, update)
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
	if err.Error() != "document not found" {
		t.Errorf("Expected 'document not found' error, got: %v", err)
	}
}

func TestRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	doc := createTestDocument(userID)

	// Create document first
	err := repo.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete document
	err = repo.Delete(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify document is deleted
	_, err = repo.GetByID(ctx, doc.ID)
	if err == nil {
		t.Error("Expected error when getting deleted document")
	}
}

func TestRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	nonExistentID := primitive.NewObjectID()

	err := repo.Delete(ctx, nonExistentID)
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
	if err.Error() != "document not found" {
		t.Errorf("Expected 'document not found' error, got: %v", err)
	}
}

func TestRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Create indexes first for text search
	err := repo.CreateIndexes(ctx)
	if err != nil {
		t.Fatalf("CreateIndexes failed: %v", err)
	}

	userID := primitive.NewObjectID()

	// Create test documents
	docs := []*models.Document{
		{
			Name:           "policy-document.pdf",
			Content:        "This is a policy document about government regulations.",
			ContentType:    "application/pdf",
			Size:           1024,
			UploadedBy:     userID,
			UploadedAt:     time.Now(),
			Classification: models.SecurityClassification{Level: "PUBLIC"},
			Metadata: models.DocumentMetadata{
				Category: models.DocumentCategoryPolicy,
				Tags:     []string{"policy", "regulations"},
				Language: "en",
			},
			ProcessingStatus: models.ProcessingStatusCompleted,
		},
		{
			Name:           "strategy-document.pdf",
			Content:        "This is a strategy document about technology implementation.",
			ContentType:    "application/pdf",
			Size:           2048,
			UploadedBy:     userID,
			UploadedAt:     time.Now().Add(-1 * time.Hour),
			Classification: models.SecurityClassification{Level: "INTERNAL"},
			Metadata: models.DocumentMetadata{
				Category: models.DocumentCategoryStrategy,
				Tags:     []string{"strategy", "technology"},
				Language: "en",
			},
			ProcessingStatus: models.ProcessingStatusCompleted,
		},
	}

	for _, doc := range docs {
		err := repo.Create(ctx, doc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test text search
	filter := SearchFilter{
		Query: "policy",
		Limit: 10,
	}

	results, total, err := repo.Search(ctx, filter)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if total == 0 {
		t.Error("Expected at least one result for text search")
	}
	if len(results) == 0 {
		t.Error("Expected at least one document in results")
	}

	// Test category filter
	filter = SearchFilter{
		Category: models.DocumentCategoryPolicy,
		Limit:    10,
	}

	results, total, err = repo.Search(ctx, filter)
	if err != nil {
		t.Fatalf("Search by category failed: %v", err)
	}

	if total == 0 {
		t.Error("Expected at least one result for category search")
	}

	// Test tags filter
	filter = SearchFilter{
		Tags:  []string{"technology"},
		Limit: 10,
	}

	results, total, err = repo.Search(ctx, filter)
	if err != nil {
		t.Fatalf("Search by tags failed: %v", err)
	}

	if total == 0 {
		t.Error("Expected at least one result for tags search")
	}

	// Test user filter
	filter = SearchFilter{
		UploadedBy: &userID,
		Limit:      10,
	}

	results, total, err = repo.Search(ctx, filter)
	if err != nil {
		t.Fatalf("Search by user failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 results for user search, got %d", total)
	}
}

func TestRepository_GetByStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()

	// Create documents with different statuses
	statuses := []models.ProcessingStatus{
		models.ProcessingStatusPending,
		models.ProcessingStatusProcessing,
		models.ProcessingStatusCompleted,
		models.ProcessingStatusFailed,
	}

	for i, status := range statuses {
		doc := createTestDocument(userID)
		doc.Name = fmt.Sprintf("doc-%d.pdf", i)
		doc.ProcessingStatus = status
		err := repo.Create(ctx, doc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test getting documents by status
	results, err := repo.GetByStatus(ctx, models.ProcessingStatusPending, 10)
	if err != nil {
		t.Fatalf("GetByStatus failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 pending document, got %d", len(results))
	}

	if results[0].ProcessingStatus != models.ProcessingStatusPending {
		t.Errorf("Expected status %s, got %s", models.ProcessingStatusPending, results[0].ProcessingStatus)
	}
}

func TestRepository_GetByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID1 := primitive.NewObjectID()
	userID2 := primitive.NewObjectID()

	// Create documents for different users
	for i := 0; i < 3; i++ {
		doc := createTestDocument(userID1)
		doc.Name = fmt.Sprintf("user1-doc-%d.pdf", i)
		err := repo.Create(ctx, doc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		doc := createTestDocument(userID2)
		doc.Name = fmt.Sprintf("user2-doc-%d.pdf", i)
		err := repo.Create(ctx, doc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test getting documents by user
	results, total, err := repo.GetByUser(ctx, userID1, 10, 0)
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected 3 documents for user1, got %d", total)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 documents in results, got %d", len(results))
	}

	// Test pagination
	results, total, err = repo.GetByUser(ctx, userID1, 2, 1)
	if err != nil {
		t.Fatalf("GetByUser with pagination failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total of 3 documents, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 documents in paginated results, got %d", len(results))
	}
}

func TestRepository_UpdateProcessingStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	doc := createTestDocument(userID)

	// Create document
	err := repo.Create(ctx, doc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update processing status
	err = repo.UpdateProcessingStatus(ctx, doc.ID, models.ProcessingStatusCompleted, "")
	if err != nil {
		t.Fatalf("UpdateProcessingStatus failed: %v", err)
	}

	// Verify update
	result, err := repo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if result.ProcessingStatus != models.ProcessingStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ProcessingStatusCompleted, result.ProcessingStatus)
	}

	// Test with error message
	errorMsg := "Processing failed due to invalid format"
	err = repo.UpdateProcessingStatus(ctx, doc.ID, models.ProcessingStatusFailed, errorMsg)
	if err != nil {
		t.Fatalf("UpdateProcessingStatus with error failed: %v", err)
	}

	result, err = repo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if result.ProcessingStatus != models.ProcessingStatusFailed {
		t.Errorf("Expected status %s, got %s", models.ProcessingStatusFailed, result.ProcessingStatus)
	}
	if result.ProcessingError == nil || *result.ProcessingError != errorMsg {
		t.Errorf("Expected error message '%s', got %v", errorMsg, result.ProcessingError)
	}
}

func TestRepository_GetStatistics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()

	// Create documents with different statuses and categories
	testData := []struct {
		status   models.ProcessingStatus
		category models.DocumentCategory
	}{
		{models.ProcessingStatusPending, models.DocumentCategoryPolicy},
		{models.ProcessingStatusCompleted, models.DocumentCategoryPolicy},
		{models.ProcessingStatusCompleted, models.DocumentCategoryStrategy},
		{models.ProcessingStatusFailed, models.DocumentCategoryGeneral},
	}

	for i, data := range testData {
		doc := createTestDocument(userID)
		doc.Name = fmt.Sprintf("stats-doc-%d.pdf", i)
		doc.ProcessingStatus = data.status
		doc.Metadata.Category = data.category
		err := repo.Create(ctx, doc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Get statistics
	stats, err := repo.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	// Verify total count
	total, ok := stats["total"].(int64)
	if !ok {
		t.Error("Expected total to be int64")
	}
	if total != 4 {
		t.Errorf("Expected total of 4 documents, got %d", total)
	}

	// Verify status statistics
	statusStats, ok := stats["by_status"].(map[string]int64)
	if !ok {
		t.Error("Expected by_status to be map[string]int64")
	}

	expectedStatusCounts := map[string]int64{
		string(models.ProcessingStatusPending):   1,
		string(models.ProcessingStatusCompleted): 2,
		string(models.ProcessingStatusFailed):    1,
	}

	for status, expectedCount := range expectedStatusCounts {
		if count, exists := statusStats[status]; !exists || count != expectedCount {
			t.Errorf("Expected %d documents with status %s, got %d", expectedCount, status, count)
		}
	}

	// Verify category statistics
	categoryStats, ok := stats["by_category"].(map[string]int64)
	if !ok {
		t.Error("Expected by_category to be map[string]int64")
	}

	expectedCategoryCounts := map[string]int64{
		string(models.DocumentCategoryPolicy):   2,
		string(models.DocumentCategoryStrategy): 1,
		string(models.DocumentCategoryGeneral):  1,
	}

	for category, expectedCount := range expectedCategoryCounts {
		if count, exists := categoryStats[category]; !exists || count != expectedCount {
			t.Errorf("Expected %d documents with category %s, got %d", expectedCount, category, count)
		}
	}
}

func TestRepository_SearchFilter_DateRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	userID := primitive.NewObjectID()

	// Create documents with different upload dates
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	docs := []*models.Document{
		{
			Name:             "old-doc.pdf",
			Content:          "Old document",
			ContentType:      "application/pdf",
			Size:             1024,
			UploadedBy:       userID,
			UploadedAt:       yesterday,
			Classification:   models.SecurityClassification{Level: "PUBLIC"},
			Metadata:         models.DocumentMetadata{Category: models.DocumentCategoryGeneral, Language: "en"},
			ProcessingStatus: models.ProcessingStatusCompleted,
		},
		{
			Name:             "current-doc.pdf",
			Content:          "Current document",
			ContentType:      "application/pdf",
			Size:             1024,
			UploadedBy:       userID,
			UploadedAt:       now,
			Classification:   models.SecurityClassification{Level: "PUBLIC"},
			Metadata:         models.DocumentMetadata{Category: models.DocumentCategoryGeneral, Language: "en"},
			ProcessingStatus: models.ProcessingStatusCompleted,
		},
		{
			Name:             "future-doc.pdf",
			Content:          "Future document",
			ContentType:      "application/pdf",
			Size:             1024,
			UploadedBy:       userID,
			UploadedAt:       tomorrow,
			Classification:   models.SecurityClassification{Level: "PUBLIC"},
			Metadata:         models.DocumentMetadata{Category: models.DocumentCategoryGeneral, Language: "en"},
			ProcessingStatus: models.ProcessingStatusCompleted,
		},
	}

	for _, doc := range docs {
		err := repo.Create(ctx, doc)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test date range filter
	filter := SearchFilter{
		DateFrom: &yesterday,
		DateTo:   &now,
		Limit:    10,
	}

	results, total, err := repo.Search(ctx, filter)
	if err != nil {
		t.Fatalf("Search with date range failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 documents in date range, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 documents in results, got %d", len(results))
	}
}
