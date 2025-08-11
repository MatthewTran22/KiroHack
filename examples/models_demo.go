package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"ai-government-consultant/internal/database"
	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	fmt.Println("AI Government Consultant - Data Models Demo")
	fmt.Println("==========================================")

	// Demo 1: Create and validate data models
	demoDataModels()

	// Demo 2: MongoDB connection (will fail without running MongoDB)
	demoMongoDBConnection()
}

func demoDataModels() {
	fmt.Println("\n1. Data Models Demo")
	fmt.Println("-------------------")

	// Create a user
	user := models.User{
		ID:                primitive.NewObjectID(),
		Email:             "analyst@government.gov",
		Name:              "Jane Analyst",
		Department:        "Policy Analysis",
		Role:              models.UserRoleAnalyst,
		SecurityClearance: models.SecurityClearanceConfidential,
		Permissions: []models.Permission{
			{
				Resource: "documents",
				Actions:  []string{"read", "write"},
			},
			{
				Resource: "consultations",
				Actions:  []string{"read"},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}

	// Validate user
	if err := user.Validate(); err != nil {
		log.Printf("User validation failed: %v", err)
	} else {
		fmt.Printf("✓ User created and validated: %s (%s)\n", user.Name, user.Email)
	}

	// Test user permissions
	fmt.Printf("✓ User can read documents: %v\n", user.HasPermission("documents", "read"))
	fmt.Printf("✓ User can delete documents: %v\n", user.HasPermission("documents", "delete"))
	fmt.Printf("✓ User can access CONFIDENTIAL docs: %v\n", user.CanAccessClassification("CONFIDENTIAL"))
	fmt.Printf("✓ User can access SECRET docs: %v\n", user.CanAccessClassification("SECRET"))

	// Create a document
	document := models.Document{
		ID:          primitive.NewObjectID(),
		Name:        "policy-analysis-2024.pdf",
		Content:     "This document contains policy analysis for 2024...",
		ContentType: "application/pdf",
		Size:        1024000,
		UploadedBy:  user.ID,
		UploadedAt:  time.Now(),
		Classification: models.SecurityClassification{
			Level:        "CONFIDENTIAL",
			Compartments: []string{"NOFORN"},
			Handling:     []string{"CONTROLLED"},
		},
		Metadata: models.DocumentMetadata{
			Title:      stringPtr("Policy Analysis 2024"),
			Author:     stringPtr("Policy Team"),
			Department: stringPtr("Policy Analysis"),
			Category:   models.DocumentCategoryPolicy,
			Tags:       []string{"policy", "analysis", "2024"},
			Language:   "en",
		},
		ProcessingStatus: models.ProcessingStatusCompleted,
		Embeddings:       []float64{0.1, 0.2, 0.3, 0.4, 0.5},
	}

	// Validate document
	if err := document.Validate(); err != nil {
		log.Printf("Document validation failed: %v", err)
	} else {
		fmt.Printf("✓ Document created and validated: %s\n", document.Name)
	}

	fmt.Printf("✓ Document is processed: %v\n", document.IsProcessed())
	fmt.Printf("✓ Document has embeddings: %v\n", document.HasEmbeddings())

	// Create a knowledge item
	knowledge := models.KnowledgeItem{
		ID:       primitive.NewObjectID(),
		Content:  "Government agencies must follow FISMA requirements for information security management.",
		Type:     models.KnowledgeTypeRegulation,
		Title:    "FISMA Compliance Requirements",
		Summary:  stringPtr("Federal agencies must implement FISMA security controls"),
		Keywords: []string{"FISMA", "security", "compliance"},
		Tags:     []string{"security", "regulation"},
		Category: "Information Security",
		Source: models.KnowledgeSource{
			Type:        "document",
			SourceID:    document.ID,
			Reference:   "FISMA Act of 2002",
			Reliability: 1.0,
		},
		Confidence: 0.95,
		Validation: models.KnowledgeValidation{
			IsValidated: true,
			ValidatedBy: &user.ID,
			ValidatedAt: timePtr(time.Now()),
		},
		Usage: models.KnowledgeUsage{
			AccessCount:        0,
			UsageContexts:      []string{},
			EffectivenessScore: 0.0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: user.ID,
		Version:   1,
		IsActive:  true,
		Metadata:  make(map[string]interface{}),
	}

	// Validate knowledge item
	if err := knowledge.Validate(); err != nil {
		log.Printf("Knowledge item validation failed: %v", err)
	} else {
		fmt.Printf("✓ Knowledge item created and validated: %s\n", knowledge.Title)
	}

	fmt.Printf("✓ Knowledge item is validated: %v\n", knowledge.IsValidated())
	fmt.Printf("✓ Knowledge item is expired: %v\n", knowledge.IsExpired())

	// Test knowledge item usage
	knowledge.IncrementUsage("policy_analysis")
	fmt.Printf("✓ Knowledge item access count after usage: %d\n", knowledge.Usage.AccessCount)

	// Create a consultation session
	consultation := models.ConsultationSession{
		ID:     primitive.NewObjectID(),
		UserID: user.ID,
		Type:   models.ConsultationTypePolicy,
		Query:  "What are the key FISMA compliance requirements for our agency?",
		Context: models.ConsultationContext{
			RelatedDocuments: []primitive.ObjectID{document.ID},
			UserContext: map[string]interface{}{
				"department": user.Department,
				"clearance":  user.SecurityClearance,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    models.SessionStatusActive,
		Tags:      []string{"fisma", "compliance"},
	}

	// Validate consultation
	if err := consultation.Validate(); err != nil {
		log.Printf("Consultation validation failed: %v", err)
	} else {
		fmt.Printf("✓ Consultation session created and validated\n")
	}

	fmt.Printf("✓ Consultation is completed: %v\n", consultation.IsCompleted())
	fmt.Printf("✓ Consultation has response: %v\n", consultation.HasResponse())
}

func demoMongoDBConnection() {
	fmt.Println("\n2. MongoDB Connection Demo")
	fmt.Println("--------------------------")

	// Create MongoDB configuration for Docker setup
	config := &database.Config{
		URI:            "mongodb://admin:password@localhost:27017/ai_government_consultant?authSource=admin",
		DatabaseName:   "ai_government_consultant",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    5 * time.Minute,
	}
	fmt.Printf("✓ MongoDB config created for Docker setup: %s\n", config.DatabaseName)

	// Attempt to connect to MongoDB
	fmt.Println("Attempting to connect to MongoDB...")
	mongodb, err := database.NewMongoDB(config)
	if err != nil {
		fmt.Printf("✗ MongoDB connection failed (expected if MongoDB is not running): %v\n", err)
		return
	}
	defer func() {
		ctx := context.Background()
		mongodb.Close(ctx)
	}()

	fmt.Println("✓ MongoDB connection successful!")

	// Test ping
	ctx := context.Background()
	if err := mongodb.Ping(ctx); err != nil {
		fmt.Printf("✗ MongoDB ping failed: %v\n", err)
		return
	}
	fmt.Println("✓ MongoDB ping successful!")

	// Test health check
	if err := mongodb.HealthCheck(ctx); err != nil {
		fmt.Printf("✗ MongoDB health check failed: %v\n", err)
		return
	}
	fmt.Println("✓ MongoDB health check successful!")

	// Create indexes
	if err := mongodb.CreateIndexes(ctx); err != nil {
		fmt.Printf("✗ Index creation failed: %v\n", err)
		return
	}
	fmt.Println("✓ Database indexes created successfully!")

	// Initialize database with sample data
	if err := database.InitializeDatabase(mongodb); err != nil {
		fmt.Printf("✗ Database initialization failed: %v\n", err)
		return
	}
	fmt.Println("✓ Database initialized with sample data!")
}

func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
