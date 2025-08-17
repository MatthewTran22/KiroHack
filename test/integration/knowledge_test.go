package integration

import (
	"context"
	"testing"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/knowledge"
	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestKnowledgeManagementIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup database connection to Docker container
	config := &database.Config{
		URI:            "mongodb://admin:password@localhost:27017/test_integration?authSource=admin",
		DatabaseName:   "test_integration",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    1 * time.Minute,
	}

	mongodb, err := database.NewMongoDB(config)
	if err != nil {
		t.Skipf("Skipping integration test - MongoDB Docker container not available: %v", err)
	}

	// Clean up test data
	ctx := context.Background()
	defer func() {
		mongodb.Database.Drop(ctx)
		mongodb.Close(ctx)
	}()

	testLogger := logger.NewTestLogger()
	service := knowledge.NewService(mongodb.Database, testLogger)

	// Initialize indexes
	err = service.GetRepository().CreateIndexes(ctx)
	if err != nil {
		t.Fatalf("Failed to create indexes: %v", err)
	}

	t.Run("Complete Knowledge Lifecycle", func(t *testing.T) {
		testCompleteKnowledgeLifecycle(t, service)
	})

	t.Run("Knowledge Relationships", func(t *testing.T) {
		testKnowledgeRelationships(t, service)
	})

	t.Run("Knowledge Search and Filtering", func(t *testing.T) {
		testKnowledgeSearchAndFiltering(t, service)
	})

	t.Run("Knowledge Validation and Consistency", func(t *testing.T) {
		testKnowledgeValidationAndConsistency(t, service)
	})

	t.Run("Knowledge Graph Construction", func(t *testing.T) {
		testKnowledgeGraphConstruction(t, service)
	})

	t.Run("Document Knowledge Extraction", func(t *testing.T) {
		testDocumentKnowledgeExtraction(t, service)
	})

	t.Run("Knowledge Export and Import", func(t *testing.T) {
		testKnowledgeExportImport(t, service)
	})
}

func testCompleteKnowledgeLifecycle(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create knowledge item
	item := &models.KnowledgeItem{
		Content:    "Government agencies must implement FISMA security controls to protect federal information systems.",
		Type:       models.KnowledgeTypeRegulation,
		Title:      "FISMA Security Control Requirements",
		Category:   "Information Security",
		Tags:       []string{"FISMA", "security", "compliance"},
		Keywords:   []string{"FISMA", "security", "controls", "federal"},
		Source: models.KnowledgeSource{
			Type:        "manual",
			SourceID:    primitive.NewObjectID(),
			Reference:   "FISMA Act Section 3544",
			Reliability: 1.0,
		},
		Confidence: 0.95,
		CreatedBy:  createdBy,
		Validation: models.KnowledgeValidation{
			IsValidated: false,
		},
		Usage: models.KnowledgeUsage{
			AccessCount:        0,
			UsageContexts:      []string{},
			EffectivenessScore: 0.0,
		},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	// Test creation
	createdItem, err := service.CreateKnowledgeItem(ctx, item)
	if err != nil {
		t.Fatalf("Failed to create knowledge item: %v", err)
	}

	if createdItem.ID.IsZero() {
		t.Error("Created item should have an ID")
	}

	if createdItem.Version != 1 {
		t.Errorf("Created item version = %d, want 1", createdItem.Version)
	}

	// Test retrieval
	retrievedItem, err := service.GetKnowledgeItem(ctx, createdItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve knowledge item: %v", err)
	}

	if retrievedItem.Title != item.Title {
		t.Errorf("Retrieved item title = %v, want %v", retrievedItem.Title, item.Title)
	}

	// Test update
	updates := map[string]interface{}{
		"confidence": 0.98,
		"content":    "Updated: Government agencies must implement comprehensive FISMA security controls to protect federal information systems and ensure compliance.",
	}

	updatedItem, err := service.UpdateKnowledgeItem(ctx, createdItem.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update knowledge item: %v", err)
	}

	if updatedItem.Confidence != 0.98 {
		t.Errorf("Updated item confidence = %v, want 0.98", updatedItem.Confidence)
	}

	if updatedItem.Version != 2 {
		t.Errorf("Updated item version = %d, want 2", updatedItem.Version)
	}

	// Test validation
	validatedBy := primitive.NewObjectID()
	expiresAt := time.Now().Add(365 * 24 * time.Hour) // 1 year from now
	err = service.ValidateKnowledgeItem(ctx, createdItem.ID, validatedBy, "Validated against official FISMA documentation", &expiresAt)
	if err != nil {
		t.Fatalf("Failed to validate knowledge item: %v", err)
	}

	validatedItem, err := service.GetKnowledgeItem(ctx, createdItem.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve validated item: %v", err)
	}

	if !validatedItem.Validation.IsValidated {
		t.Error("Item should be validated")
	}

	if validatedItem.Validation.ValidatedBy == nil || *validatedItem.Validation.ValidatedBy != validatedBy {
		t.Error("Item should have correct validated by user")
	}

	// Test versioning
	versionedItem, err := service.CreateKnowledgeVersion(ctx, createdItem.ID, map[string]interface{}{
		"content": "Version 2: Government agencies must implement comprehensive FISMA security controls with regular assessments.",
	}, validatedBy, "major", []string{"Added requirement for regular assessments"})
	if err != nil {
		t.Fatalf("Failed to create knowledge version: %v", err)
	}

	if versionedItem.Version <= validatedItem.Version {
		t.Error("Versioned item should have higher version number")
	}

	// Test version history
	versionHistory, err := service.GetKnowledgeVersionHistory(ctx, createdItem.ID)
	if err != nil {
		t.Fatalf("Failed to get version history: %v", err)
	}

	if len(versionHistory) == 0 {
		t.Error("Should have version history")
	}

	// Test soft delete
	err = service.DeleteKnowledgeItem(ctx, createdItem.ID)
	if err != nil {
		t.Fatalf("Failed to delete knowledge item: %v", err)
	}

	// Verify item is soft deleted (not accessible via normal get)
	_, err = service.GetKnowledgeItem(ctx, createdItem.ID)
	if err == nil {
		t.Error("Deleted item should not be accessible")
	}
}

func testKnowledgeRelationships(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create related knowledge items
	policyItem := &models.KnowledgeItem{
		Content:       "Security policies must be established and maintained.",
		Type:          models.KnowledgeTypeRule,
		Title:         "Security Policy Requirement",
		Category:      "Information Security",
		Tags:          []string{"security", "policy"},
		Keywords:      []string{"security", "policy"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Security Framework", Reliability: 0.9},
		Confidence:    0.9,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	procedureItem := &models.KnowledgeItem{
		Content:       "Follow these steps to implement security policies: 1. Assess current state, 2. Define requirements, 3. Implement controls.",
		Type:          models.KnowledgeTypeProcedure,
		Title:         "Security Policy Implementation Procedure",
		Category:      "Information Security",
		Tags:          []string{"security", "procedure", "implementation"},
		Keywords:      []string{"security", "procedure", "implementation"},
		Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Implementation Guide", Reliability: 0.85},
		Confidence:    0.85,
		CreatedBy:     createdBy,
		Validation:    models.KnowledgeValidation{IsValidated: false},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	// Create items
	createdPolicy, err := service.CreateKnowledgeItem(ctx, policyItem)
	if err != nil {
		t.Fatalf("Failed to create policy item: %v", err)
	}

	createdProcedure, err := service.CreateKnowledgeItem(ctx, procedureItem)
	if err != nil {
		t.Fatalf("Failed to create procedure item: %v", err)
	}

	// Test adding relationship
	err = service.AddRelationship(ctx, createdPolicy.ID, createdProcedure.ID, models.RelationshipTypeImplements, 0.9, "procedure implements policy")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Test getting related items
	relatedItems, err := service.GetRelatedKnowledge(ctx, createdProcedure.ID, models.RelationshipTypeImplements, 10)
	if err != nil {
		t.Fatalf("Failed to get related items: %v", err)
	}

	if len(relatedItems) != 1 {
		t.Errorf("Related items count = %d, want 1", len(relatedItems))
	}

	if len(relatedItems) > 0 && relatedItems[0].ID != createdPolicy.ID {
		t.Error("Related item should be the policy item")
	}

	// Test removing relationship
	err = service.RemoveRelationship(ctx, createdPolicy.ID, createdProcedure.ID, models.RelationshipTypeImplements)
	if err != nil {
		t.Fatalf("Failed to remove relationship: %v", err)
	}

	// Verify relationship was removed
	relatedItemsAfterRemoval, err := service.GetRelatedKnowledge(ctx, createdProcedure.ID, models.RelationshipTypeImplements, 10)
	if err != nil {
		t.Fatalf("Failed to get related items after removal: %v", err)
	}

	if len(relatedItemsAfterRemoval) != 0 {
		t.Errorf("Related items count after removal = %d, want 0", len(relatedItemsAfterRemoval))
	}
}

func testKnowledgeSearchAndFiltering(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create diverse knowledge items for testing search
	items := []*models.KnowledgeItem{
		{
			Content:       "FISMA requires federal agencies to implement security controls",
			Type:          models.KnowledgeTypeRegulation,
			Title:         "FISMA Security Requirements",
			Category:      "Information Security",
			Tags:          []string{"FISMA", "security", "federal"},
			Keywords:      []string{"FISMA", "security", "controls"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "FISMA Act", Reliability: 1.0},
			Confidence:    0.95,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: true},
			Usage:         models.KnowledgeUsage{AccessCount: 10, UsageContexts: []string{"compliance"}, EffectivenessScore: 0.9},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Best practice: Use multi-factor authentication for all administrative accounts",
			Type:          models.KnowledgeTypeBestPractice,
			Title:         "MFA Best Practice",
			Category:      "Information Security",
			Tags:          []string{"MFA", "authentication", "security"},
			Keywords:      []string{"MFA", "authentication", "administrative"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Security Guide", Reliability: 0.9},
			Confidence:    0.85,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 5, UsageContexts: []string{"security"}, EffectivenessScore: 0.8},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Policy development requires stakeholder consultation and impact assessment",
			Type:          models.KnowledgeTypeProcedure,
			Title:         "Policy Development Process",
			Category:      "Policy Management",
			Tags:          []string{"policy", "development", "consultation"},
			Keywords:      []string{"policy", "stakeholder", "assessment"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Policy Guide", Reliability: 0.8},
			Confidence:    0.8,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: true},
			Usage:         models.KnowledgeUsage{AccessCount: 3, UsageContexts: []string{"policy"}, EffectivenessScore: 0.7},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	// Create all items
	var createdItems []*models.KnowledgeItem
	for _, item := range items {
		createdItem, err := service.CreateKnowledgeItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}
		createdItems = append(createdItems, createdItem)
	}

	// Test search by type
	typeFilter := knowledge.SearchFilter{
		Type:  models.KnowledgeTypeRegulation,
		Limit: 10,
	}
	typeResults, total, err := service.SearchKnowledge(ctx, typeFilter)
	if err != nil {
		t.Fatalf("Failed to search by type: %v", err)
	}

	if len(typeResults) != 1 {
		t.Errorf("Type search results = %d, want 1", len(typeResults))
	}

	if total != 1 {
		t.Errorf("Type search total = %d, want 1", total)
	}

	// Test search by category
	categoryFilter := knowledge.SearchFilter{
		Category: "Information Security",
		Limit:    10,
	}
	categoryResults, _, err := service.SearchKnowledge(ctx, categoryFilter)
	if err != nil {
		t.Fatalf("Failed to search by category: %v", err)
	}

	if len(categoryResults) != 2 {
		t.Errorf("Category search results = %d, want 2", len(categoryResults))
	}

	// Test search by confidence
	confidenceFilter := knowledge.SearchFilter{
		MinConfidence: 0.9,
		Limit:         10,
	}
	confidenceResults, _, err := service.SearchKnowledge(ctx, confidenceFilter)
	if err != nil {
		t.Fatalf("Failed to search by confidence: %v", err)
	}

	if len(confidenceResults) != 1 {
		t.Errorf("Confidence search results = %d, want 1", len(confidenceResults))
	}

	// Test search by tags
	tagFilter := knowledge.SearchFilter{
		Tags:  []string{"security"},
		Limit: 10,
	}
	tagResults, _, err := service.SearchKnowledge(ctx, tagFilter)
	if err != nil {
		t.Fatalf("Failed to search by tags: %v", err)
	}

	if len(tagResults) != 2 {
		t.Errorf("Tag search results = %d, want 2", len(tagResults))
	}

	// Test search by validation status
	validatedTrue := true
	validationFilter := knowledge.SearchFilter{
		IsValidated: &validatedTrue,
		Limit:       10,
	}
	validationResults, _, err := service.SearchKnowledge(ctx, validationFilter)
	if err != nil {
		t.Fatalf("Failed to search by validation: %v", err)
	}

	if len(validationResults) != 2 {
		t.Errorf("Validation search results = %d, want 2", len(validationResults))
	}

	// Test search with limit
	limitFilter := knowledge.SearchFilter{
		Limit: 2,
	}
	limitResults, _, err := service.SearchKnowledge(ctx, limitFilter)
	if err != nil {
		t.Fatalf("Failed to search with limit: %v", err)
	}

	if len(limitResults) != 2 {
		t.Errorf("Limit search results = %d, want 2", len(limitResults))
	}
}

func testKnowledgeValidationAndConsistency(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create items with potential consistency issues
	expiredItem := &models.KnowledgeItem{
		Content:    "This knowledge has expired",
		Type:       models.KnowledgeTypeFact,
		Title:      "Expired Knowledge",
		Category:   "Test Category",
		Tags:       []string{"test", "expired"},
		Keywords:   []string{"test", "expired"},
		Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.8},
		Confidence: 0.8,
		CreatedBy:  createdBy,
		Validation: models.KnowledgeValidation{
			IsValidated: true,
			ExpiresAt:   timePtr(time.Now().Add(-24 * time.Hour)), // Expired 1 day ago
		},
		Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	highUsageLowConfidenceItem := &models.KnowledgeItem{
		Content:    "This knowledge has high usage but low confidence",
		Type:       models.KnowledgeTypeFact,
		Title:      "High Usage Low Confidence",
		Category:   "Test Category",
		Tags:       []string{"test", "usage"},
		Keywords:   []string{"test", "usage"},
		Source:     models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.5},
		Confidence: 0.4, // Low confidence
		CreatedBy:  createdBy,
		Validation: models.KnowledgeValidation{IsValidated: false},
		Usage:      models.KnowledgeUsage{AccessCount: 15, UsageContexts: []string{"test"}, EffectivenessScore: 0.3}, // High usage
		Relationships: []models.KnowledgeRelationship{},
		Metadata:      make(map[string]interface{}),
	}

	// Create items
	_, err := service.CreateKnowledgeItem(ctx, expiredItem)
	if err != nil {
		t.Fatalf("Failed to create expired item: %v", err)
	}

	_, err = service.CreateKnowledgeItem(ctx, highUsageLowConfidenceItem)
	if err != nil {
		t.Fatalf("Failed to create high usage item: %v", err)
	}

	// Test consistency validation
	issues, err := service.ValidateConsistency(ctx)
	if err != nil {
		t.Fatalf("Failed to validate consistency: %v", err)
	}

	if len(issues) < 2 {
		t.Errorf("Consistency validation should find at least 2 issues, found %d", len(issues))
	}

	// Check for specific issue types
	foundExpired := false
	foundHighUsageLowConfidence := false

	for _, issue := range issues {
		switch issue.Type {
		case "expired":
			foundExpired = true
			if issue.Severity != "medium" {
				t.Errorf("Expired issue severity = %v, want 'medium'", issue.Severity)
			}
		case "low_confidence_high_usage":
			foundHighUsageLowConfidence = true
			if issue.Severity != "medium" {
				t.Errorf("High usage low confidence issue severity = %v, want 'medium'", issue.Severity)
			}
		}
	}

	if !foundExpired {
		t.Error("Should find expired knowledge issue")
	}

	if !foundHighUsageLowConfidence {
		t.Error("Should find high usage low confidence issue")
	}

	// Test getting recommendations
	recommendations, err := service.GetKnowledgeRecommendations(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get recommendations: %v", err)
	}

	if len(recommendations) == 0 {
		t.Error("Should get at least one recommendation")
	}

	// Check for specific recommendation types
	foundValidationRec := false
	foundUpdateRec := false

	for _, rec := range recommendations {
		switch rec.Type {
		case "validation":
			foundValidationRec = true
		case "update":
			foundUpdateRec = true
		}
	}

	if !foundValidationRec {
		t.Error("Should get validation recommendation for expired item")
	}

	if !foundUpdateRec {
		t.Error("Should get update recommendation for high usage low confidence item")
	}
}

func testKnowledgeGraphConstruction(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create items for graph construction
	items := []*models.KnowledgeItem{
		{
			Content:       "Security policy must be established",
			Type:          models.KnowledgeTypeRule,
			Title:         "Security Policy Rule",
			Category:      "Information Security",
			Tags:          []string{"security", "policy"},
			Keywords:      []string{"security", "policy"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Security Framework", Reliability: 0.9},
			Confidence:    0.9,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Implement security policy through these procedures",
			Type:          models.KnowledgeTypeProcedure,
			Title:         "Security Policy Implementation",
			Category:      "Information Security",
			Tags:          []string{"security", "implementation"},
			Keywords:      []string{"security", "implementation"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Implementation Guide", Reliability: 0.85},
			Confidence:    0.85,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 0, UsageContexts: []string{}, EffectivenessScore: 0.0},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	// Create items
	var createdItems []*models.KnowledgeItem
	for _, item := range items {
		createdItem, err := service.CreateKnowledgeItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create item for graph: %v", err)
		}
		createdItems = append(createdItems, createdItem)
	}

	// Add relationship
	err := service.AddRelationship(ctx, createdItems[0].ID, createdItems[1].ID, models.RelationshipTypeImplements, 0.9, "procedure implements rule")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Test building knowledge graph
	err = service.BuildKnowledgeGraph(ctx)
	if err != nil {
		t.Fatalf("Failed to build knowledge graph: %v", err)
	}

	// Test getting knowledge graph
	filter := knowledge.SearchFilter{
		Category: "Information Security",
		Limit:    10,
	}
	graph, err := service.GetKnowledgeGraph(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get knowledge graph: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("Graph nodes = %d, want 2", len(graph.Nodes))
	}

	if len(graph.Edges) == 0 {
		t.Error("Graph should have at least one edge")
	}

	// Verify node properties
	for _, node := range graph.Nodes {
		if node.ID == "" {
			t.Error("Graph node should have ID")
		}
		if node.Title == "" {
			t.Error("Graph node should have title")
		}
		if node.Category != "Information Security" {
			t.Errorf("Graph node category = %v, want 'Information Security'", node.Category)
		}
	}

	// Verify edge properties
	for _, edge := range graph.Edges {
		if edge.Source == "" || edge.Target == "" {
			t.Error("Graph edge should have source and target")
		}
		if edge.Strength <= 0 {
			t.Error("Graph edge should have positive strength")
		}
	}
}

func testDocumentKnowledgeExtraction(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	extractedBy := primitive.NewObjectID()

	// Create test document
	document := &models.Document{
		ID:   primitive.NewObjectID(),
		Name: "Security Policy Document",
		Content: `
			Government agencies must implement comprehensive security controls to protect federal information systems.
			
			The procedure for implementing security controls requires the following steps:
			First, conduct a thorough risk assessment of all information systems.
			Second, select appropriate security controls based on the risk assessment results.
			Third, implement the selected controls according to established procedures.
			
			It is recommended to use industry best practices for security control implementation.
			Organizations should consider using automated tools for continuous monitoring.
			
			Security policies shall be reviewed annually and updated as necessary.
			All personnel must receive security awareness training.
		`,
		Metadata: models.DocumentMetadata{
			Category: models.DocumentCategoryPolicy,
			Tags:     []string{"security", "policy", "controls"},
		},
	}

	// Test knowledge extraction
	extractedItems, err := service.ExtractKnowledgeFromDocument(ctx, document, extractedBy)
	if err != nil {
		t.Fatalf("Failed to extract knowledge from document: %v", err)
	}

	if len(extractedItems) == 0 {
		t.Error("Should extract at least one knowledge item")
	}

	// Verify extracted items
	foundTypes := make(map[models.KnowledgeType]int)
	for _, item := range extractedItems {
		// Check basic properties
		if item.Source.Type != "document" {
			t.Errorf("Extracted item source type = %v, want 'document'", item.Source.Type)
		}
		if item.Source.SourceID != document.ID {
			t.Errorf("Extracted item source ID = %v, want %v", item.Source.SourceID, document.ID)
		}
		if item.CreatedBy != extractedBy {
			t.Errorf("Extracted item created by = %v, want %v", item.CreatedBy, extractedBy)
		}
		if item.Category != string(document.Metadata.Category) {
			t.Errorf("Extracted item category = %v, want %v", item.Category, string(document.Metadata.Category))
		}

		// Count types
		foundTypes[item.Type]++

		// Check content is not empty
		if item.Content == "" {
			t.Error("Extracted item should have content")
		}

		// Check title is generated
		if item.Title == "" {
			t.Error("Extracted item should have title")
		}

		// Check confidence is reasonable
		if item.Confidence <= 0 || item.Confidence > 1 {
			t.Errorf("Extracted item confidence = %v, should be between 0 and 1", item.Confidence)
		}
	}

	// Should extract different types of knowledge
	if len(foundTypes) < 2 {
		t.Errorf("Should extract multiple types of knowledge, found %d types", len(foundTypes))
	}

	// Check for relationships between extracted items
	if len(extractedItems) > 1 {
		hasRelationships := false
		for _, item := range extractedItems {
			if len(item.Relationships) > 0 {
				hasRelationships = true
				break
			}
		}
		if !hasRelationships {
			t.Error("Extracted items should have relationships between them")
		}
	}
}

func testKnowledgeExportImport(t *testing.T, service *knowledge.Service) {
	ctx := context.Background()
	createdBy := primitive.NewObjectID()

	// Create test items for export
	items := []*models.KnowledgeItem{
		{
			Content:       "Export test item 1",
			Type:          models.KnowledgeTypeFact,
			Title:         "Export Test 1",
			Category:      "Test Category",
			Tags:          []string{"export", "test"},
			Keywords:      []string{"export", "test"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.9},
			Confidence:    0.9,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: true},
			Usage:         models.KnowledgeUsage{AccessCount: 5, UsageContexts: []string{"test"}, EffectivenessScore: 0.8},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
		{
			Content:       "Export test item 2",
			Type:          models.KnowledgeTypeRule,
			Title:         "Export Test 2",
			Category:      "Test Category",
			Tags:          []string{"export", "test"},
			Keywords:      []string{"export", "test"},
			Source:        models.KnowledgeSource{Type: "manual", SourceID: primitive.NewObjectID(), Reference: "Test", Reliability: 0.85},
			Confidence:    0.85,
			CreatedBy:     createdBy,
			Validation:    models.KnowledgeValidation{IsValidated: false},
			Usage:         models.KnowledgeUsage{AccessCount: 3, UsageContexts: []string{"test"}, EffectivenessScore: 0.7},
			Relationships: []models.KnowledgeRelationship{},
			Metadata:      make(map[string]interface{}),
		},
	}

	// Create items
	var createdItems []*models.KnowledgeItem
	for _, item := range items {
		createdItem, err := service.CreateKnowledgeItem(ctx, item)
		if err != nil {
			t.Fatalf("Failed to create item for export: %v", err)
		}
		createdItems = append(createdItems, createdItem)
	}

	// Add relationship
	err := service.AddRelationship(ctx, createdItems[0].ID, createdItems[1].ID, models.RelationshipTypeRelatedTo, 0.7, "test relationship")
	if err != nil {
		t.Fatalf("Failed to add relationship for export: %v", err)
	}

	// Test JSON export
	exportOptions := knowledge.KnowledgeExportOptions{
		Format:               knowledge.ExportFormatJSON,
		IncludeRelationships: true,
		IncludeMetadata:      true,
		IncludeUsageStats:    true,
		FilterByCategory:     []string{"Test Category"},
		MinConfidence:        0.0,
		ValidatedOnly:        false,
	}

	exportData, err := service.ExportKnowledge(ctx, exportOptions)
	if err != nil {
		t.Fatalf("Failed to export knowledge: %v", err)
	}

	if len(exportData) == 0 {
		t.Error("Export data should not be empty")
	}

	// Test CSV export
	csvOptions := knowledge.KnowledgeExportOptions{
		Format:        knowledge.ExportFormatCSV,
		MinConfidence: 0.0,
		ValidatedOnly: false,
	}

	csvData, err := service.ExportKnowledge(ctx, csvOptions)
	if err != nil {
		t.Fatalf("Failed to export knowledge as CSV: %v", err)
	}

	if len(csvData) == 0 {
		t.Error("CSV export data should not be empty")
	}

	// Test Markdown export
	markdownOptions := knowledge.KnowledgeExportOptions{
		Format:        knowledge.ExportFormatMarkdown,
		MinConfidence: 0.0,
		ValidatedOnly: false,
	}

	markdownData, err := service.ExportKnowledge(ctx, markdownOptions)
	if err != nil {
		t.Fatalf("Failed to export knowledge as Markdown: %v", err)
	}

	if len(markdownData) == 0 {
		t.Error("Markdown export data should not be empty")
	}

	// Test import
	importResult, err := service.ImportKnowledge(ctx, exportData, knowledge.ExportFormatJSON, createdBy)
	if err != nil {
		t.Fatalf("Failed to import knowledge: %v", err)
	}

	if importResult.TotalItems == 0 {
		t.Error("Import should process at least one item")
	}

	if importResult.ImportedItems == 0 {
		t.Error("Import should successfully import at least one item")
	}

	if importResult.ErrorItems > 0 {
		t.Errorf("Import should not have errors, got %d errors: %v", importResult.ErrorItems, importResult.Errors)
	}

	if importResult.ProcessingTime <= 0 {
		t.Error("Import should record processing time")
	}

	// Verify imported items exist
	filter := knowledge.SearchFilter{
		Category: "Test Category",
		Limit:    20,
	}
	searchResults, _, err := service.SearchKnowledge(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to search after import: %v", err)
	}

	// Should have original items plus imported items
	expectedMinItems := len(createdItems) + importResult.ImportedItems
	if len(searchResults) < expectedMinItems {
		t.Errorf("After import, should have at least %d items, got %d", expectedMinItems, len(searchResults))
	}
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}