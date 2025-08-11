package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"ai-government-consultant/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InitializeDatabase initializes the database with required collections and seed data
func InitializeDatabase(db *MongoDB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Initializing database...")

	// Create indexes
	if err := db.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	log.Println("Database indexes created successfully")

	// Create default admin user if it doesn't exist
	if err := createDefaultAdminUser(ctx, db); err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}
	log.Println("Default admin user created/verified")

	// Create sample knowledge items
	if err := createSampleKnowledgeItems(ctx, db); err != nil {
		return fmt.Errorf("failed to create sample knowledge items: %w", err)
	}
	log.Println("Sample knowledge items created")

	log.Println("Database initialization completed successfully")
	return nil
}

// createDefaultAdminUser creates a default admin user if one doesn't exist
func createDefaultAdminUser(ctx context.Context, db *MongoDB) error {
	collection := db.GetCollection("users")

	// Check if admin user already exists
	var existingUser models.User
	err := collection.FindOne(ctx, bson.M{"email": "admin@government.gov"}).Decode(&existingUser)
	if err == nil {
		// Admin user already exists
		return nil
	}
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("error checking for existing admin user: %w", err)
	}

	// Create default admin user
	now := time.Now()
	adminUser := models.User{
		ID:         primitive.NewObjectID(),
		Email:      "admin@government.gov",
		Name:       "System Administrator",
		Department: "IT Administration",
		Role:       models.UserRoleAdmin,
		Permissions: []models.Permission{
			{
				Resource: "documents",
				Actions:  []string{"read", "write", "delete", "admin"},
			},
			{
				Resource: "consultations",
				Actions:  []string{"read", "write", "delete", "admin"},
			},
			{
				Resource: "users",
				Actions:  []string{"read", "write", "delete", "admin"},
			},
			{
				Resource: "knowledge",
				Actions:  []string{"read", "write", "delete", "admin"},
			},
			{
				Resource: "system",
				Actions:  []string{"read", "write", "admin"},
			},
		},
		SecurityClearance: models.SecurityClearanceTopSecret,
		PasswordHash:      "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj8xLs9qobKC", // "admin123" - should be changed in production
		CreatedAt:         now,
		UpdatedAt:         now,
		IsActive:          true,
		MFAEnabled:        false,
	}

	if err := adminUser.Validate(); err != nil {
		return fmt.Errorf("admin user validation failed: %w", err)
	}

	_, err = collection.InsertOne(ctx, adminUser)
	if err != nil {
		return fmt.Errorf("failed to insert admin user: %w", err)
	}

	return nil
}

// createSampleKnowledgeItems creates sample knowledge items for testing and demonstration
func createSampleKnowledgeItems(ctx context.Context, db *MongoDB) error {
	collection := db.GetCollection("knowledge_items")

	// Check if knowledge items already exist
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("error counting knowledge items: %w", err)
	}
	if count > 0 {
		// Knowledge items already exist
		return nil
	}

	// Get admin user ID for created_by field
	userCollection := db.GetCollection("users")
	var adminUser models.User
	err = userCollection.FindOne(ctx, bson.M{"email": "admin@government.gov"}).Decode(&adminUser)
	if err != nil {
		return fmt.Errorf("failed to find admin user: %w", err)
	}

	now := time.Now()
	sampleKnowledgeItems := []models.KnowledgeItem{
		{
			ID:       primitive.NewObjectID(),
			Content:  "Government agencies must follow the Federal Information Security Management Act (FISMA) requirements for information security management.",
			Type:     models.KnowledgeTypeRegulation,
			Title:    "FISMA Compliance Requirements",
			Summary:  stringPtr("Federal agencies must implement FISMA security controls"),
			Keywords: []string{"FISMA", "security", "compliance", "federal"},
			Tags:     []string{"security", "regulation", "compliance"},
			Category: "Information Security",
			Source: models.KnowledgeSource{
				Type:        "manual",
				SourceID:    primitive.NewObjectID(),
				Reference:   "FISMA Act of 2002",
				Reliability: 1.0,
			},
			Confidence: 0.95,
			Validation: models.KnowledgeValidation{
				IsValidated:     true,
				ValidatedBy:     &adminUser.ID,
				ValidatedAt:     &now,
				ValidationNotes: stringPtr("Verified against official FISMA documentation"),
			},
			Usage: models.KnowledgeUsage{
				AccessCount:        0,
				UsageContexts:      []string{},
				EffectivenessScore: 0.0,
			},
			CreatedAt:     now,
			UpdatedAt:     now,
			CreatedBy:     adminUser.ID,
			Version:       1,
			IsActive:      true,
			Metadata:      make(map[string]interface{}),
			Relationships: []models.KnowledgeRelationship{},
		},
		{
			ID:       primitive.NewObjectID(),
			Content:  "Best practice for government IT projects: Implement agile development methodologies with regular stakeholder reviews and iterative delivery cycles.",
			Type:     models.KnowledgeTypeBestPractice,
			Title:    "Agile Development for Government IT",
			Summary:  stringPtr("Agile methodologies improve government IT project success rates"),
			Keywords: []string{"agile", "development", "IT", "government", "methodology"},
			Tags:     []string{"development", "best-practice", "agile"},
			Category: "Technology Implementation",
			Source: models.KnowledgeSource{
				Type:        "manual",
				SourceID:    primitive.NewObjectID(),
				Reference:   "Government IT Best Practices Guide",
				Reliability: 0.9,
			},
			Confidence: 0.85,
			Validation: models.KnowledgeValidation{
				IsValidated:     true,
				ValidatedBy:     &adminUser.ID,
				ValidatedAt:     &now,
				ValidationNotes: stringPtr("Based on successful government IT project case studies"),
			},
			Usage: models.KnowledgeUsage{
				AccessCount:        0,
				UsageContexts:      []string{},
				EffectivenessScore: 0.0,
			},
			CreatedAt:     now,
			UpdatedAt:     now,
			CreatedBy:     adminUser.ID,
			Version:       1,
			IsActive:      true,
			Metadata:      make(map[string]interface{}),
			Relationships: []models.KnowledgeRelationship{},
		},
		{
			ID:       primitive.NewObjectID(),
			Content:  "Policy development should include stakeholder consultation, impact assessment, regulatory review, and public comment periods as required by the Administrative Procedure Act.",
			Type:     models.KnowledgeTypeProcedure,
			Title:    "Government Policy Development Process",
			Summary:  stringPtr("Standard procedure for developing government policies"),
			Keywords: []string{"policy", "development", "procedure", "stakeholder", "consultation"},
			Tags:     []string{"policy", "procedure", "governance"},
			Category: "Policy Development",
			Source: models.KnowledgeSource{
				Type:        "manual",
				SourceID:    primitive.NewObjectID(),
				Reference:   "Administrative Procedure Act Guidelines",
				Reliability: 1.0,
			},
			Confidence: 0.9,
			Validation: models.KnowledgeValidation{
				IsValidated:     true,
				ValidatedBy:     &adminUser.ID,
				ValidatedAt:     &now,
				ValidationNotes: stringPtr("Verified against APA requirements"),
			},
			Usage: models.KnowledgeUsage{
				AccessCount:        0,
				UsageContexts:      []string{},
				EffectivenessScore: 0.0,
			},
			CreatedAt:     now,
			UpdatedAt:     now,
			CreatedBy:     adminUser.ID,
			Version:       1,
			IsActive:      true,
			Metadata:      make(map[string]interface{}),
			Relationships: []models.KnowledgeRelationship{},
		},
	}

	// Insert sample knowledge items
	for _, item := range sampleKnowledgeItems {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("knowledge item validation failed: %w", err)
		}

		_, err := collection.InsertOne(ctx, item)
		if err != nil {
			return fmt.Errorf("failed to insert knowledge item: %w", err)
		}
	}

	return nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// DropDatabase drops all collections in the database (use with caution)
func DropDatabase(db *MongoDB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return db.Database.Drop(ctx)
}

// MigrateDatabase performs database migrations
func MigrateDatabase(db *MongoDB, fromVersion, toVersion int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("Migrating database from version %d to %d", fromVersion, toVersion)

	// Add migration logic here based on version numbers
	// This is a placeholder for future migration needs
	switch {
	case fromVersion < 1 && toVersion >= 1:
		// Migration to version 1
		if err := db.CreateIndexes(ctx); err != nil {
			return fmt.Errorf("migration to v1 failed: %w", err)
		}
	}

	log.Printf("Database migration completed successfully")
	return nil
}
