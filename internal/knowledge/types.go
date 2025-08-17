package knowledge

import (
	"context"

	"ai-government-consultant/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RepositoryInterface defines the interface for knowledge repository operations
type RepositoryInterface interface {
	CreateIndexes(ctx context.Context) error
	Create(ctx context.Context, item *models.KnowledgeItem) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.KnowledgeItem, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	Search(ctx context.Context, filter SearchFilter) ([]*models.KnowledgeItem, int64, error)
	GetByType(ctx context.Context, knowledgeType models.KnowledgeType, limit int) ([]*models.KnowledgeItem, error)
	GetBySource(ctx context.Context, sourceType string, sourceID primitive.ObjectID, limit int) ([]*models.KnowledgeItem, error)
	GetRelatedItems(ctx context.Context, itemID primitive.ObjectID, relationshipType models.RelationshipType, limit int) ([]*models.KnowledgeItem, error)
	GetExpiredItems(ctx context.Context, limit int) ([]*models.KnowledgeItem, error)
	UpdateUsage(ctx context.Context, id primitive.ObjectID, context string) error
	AddRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType, strength float64, relationshipContext string) error
	RemoveRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType) error
	GetStatistics(ctx context.Context) (map[string]interface{}, error)
}

// ConsistencyIssue represents an issue found during consistency validation
type ConsistencyIssue struct {
	Type        string             `json:"type"`        // "contradiction", "expired", "low_confidence_high_usage", etc.
	Description string             `json:"description"` // Human-readable description of the issue
	ItemID1     primitive.ObjectID `json:"item_id_1"`   // Primary knowledge item involved
	ItemID2     primitive.ObjectID `json:"item_id_2,omitempty"` // Secondary knowledge item (for conflicts)
	Severity    string             `json:"severity"`    // "low", "medium", "high", "critical"
	Confidence  float64            `json:"confidence"`  // Confidence in the issue detection (0.0-1.0)
	Context     string             `json:"context,omitempty"` // Additional context about the issue
}

// ConflictResolution represents a resolution for a knowledge conflict
type ConflictResolution struct {
	Action            string             `json:"action"`             // "merge", "supersede", "validate", "invalidate"
	PreferredItemID   primitive.ObjectID `json:"preferred_item_id"`  // Item to keep/prefer
	SupersededItemID  primitive.ObjectID `json:"superseded_item_id"` // Item to supersede/remove
	Notes             string             `json:"notes"`              // Resolution notes
	ResolvedBy        primitive.ObjectID `json:"resolved_by"`        // User who resolved the conflict
}

// KnowledgeGraph represents the structure of knowledge relationships
type KnowledgeGraph struct {
	Nodes []KnowledgeNode `json:"nodes"`
	Edges []KnowledgeEdge `json:"edges"`
}

// KnowledgeNode represents a node in the knowledge graph
type KnowledgeNode struct {
	ID         string                `json:"id"`
	Title      string                `json:"title"`
	Type       models.KnowledgeType  `json:"type"`
	Category   string                `json:"category"`
	Confidence float64               `json:"confidence"`
	Usage      int64                 `json:"usage"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// KnowledgeEdge represents an edge (relationship) in the knowledge graph
type KnowledgeEdge struct {
	Source   string                      `json:"source"`
	Target   string                      `json:"target"`
	Type     models.RelationshipType     `json:"type"`
	Strength float64                     `json:"strength"`
	Context  string                      `json:"context"`
}

// ExtractionResult represents the result of knowledge extraction from a document
type ExtractionResult struct {
	DocumentID       primitive.ObjectID      `json:"document_id"`
	ExtractedItems   []*models.KnowledgeItem `json:"extracted_items"`
	ExtractionStats  ExtractionStats         `json:"extraction_stats"`
	ProcessingTime   int64                   `json:"processing_time_ms"`
	Errors           []string                `json:"errors,omitempty"`
}

// ExtractionStats provides statistics about knowledge extraction
type ExtractionStats struct {
	TotalSentences    int `json:"total_sentences"`
	FactsExtracted    int `json:"facts_extracted"`
	RulesExtracted    int `json:"rules_extracted"`
	ProceduresExtracted int `json:"procedures_extracted"`
	GuidelinesExtracted int `json:"guidelines_extracted"`
}

// ValidationResult represents the result of knowledge validation
type ValidationResult struct {
	ItemID          primitive.ObjectID `json:"item_id"`
	IsValid         bool               `json:"is_valid"`
	ValidationScore float64            `json:"validation_score"`
	Issues          []string           `json:"issues,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// KnowledgeMetrics represents metrics about the knowledge base
type KnowledgeMetrics struct {
	TotalItems          int64                    `json:"total_items"`
	ItemsByType         map[string]int64         `json:"items_by_type"`
	ItemsByCategory     map[string]int64         `json:"items_by_category"`
	AverageConfidence   float64                  `json:"average_confidence"`
	ValidatedItems      int64                    `json:"validated_items"`
	ExpiredItems        int64                    `json:"expired_items"`
	TotalRelationships  int64                    `json:"total_relationships"`
	RelationshipsByType map[string]int64         `json:"relationships_by_type"`
	UsageStats          KnowledgeUsageStats      `json:"usage_stats"`
	QualityMetrics      KnowledgeQualityMetrics  `json:"quality_metrics"`
}

// KnowledgeUsageStats represents usage statistics for the knowledge base
type KnowledgeUsageStats struct {
	TotalAccesses       int64   `json:"total_accesses"`
	AverageUsageCount   float64 `json:"average_usage_count"`
	MostAccessedItems   []KnowledgeUsageItem `json:"most_accessed_items"`
	LeastAccessedItems  []KnowledgeUsageItem `json:"least_accessed_items"`
	UsageByContext      map[string]int64     `json:"usage_by_context"`
}

// KnowledgeUsageItem represents usage information for a specific knowledge item
type KnowledgeUsageItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	AccessCount int64  `json:"access_count"`
	Effectiveness float64 `json:"effectiveness"`
}

// KnowledgeQualityMetrics represents quality metrics for the knowledge base
type KnowledgeQualityMetrics struct {
	HighConfidenceItems    int64   `json:"high_confidence_items"`    // confidence > 0.8
	MediumConfidenceItems  int64   `json:"medium_confidence_items"`  // 0.5 < confidence <= 0.8
	LowConfidenceItems     int64   `json:"low_confidence_items"`     // confidence <= 0.5
	ConsistencyScore       float64 `json:"consistency_score"`        // Overall consistency score
	CompletenessScore      float64 `json:"completeness_score"`       // Coverage of knowledge domains
	FreshnessScore         float64 `json:"freshness_score"`          // How up-to-date the knowledge is
}

// SearchSuggestion represents a search suggestion based on knowledge content
type SearchSuggestion struct {
	Query       string  `json:"query"`
	Type        string  `json:"type"`        // "keyword", "phrase", "concept"
	Relevance   float64 `json:"relevance"`   // Relevance score
	Category    string  `json:"category"`    // Knowledge category
	ItemCount   int     `json:"item_count"`  // Number of items matching this suggestion
}

// KnowledgeRecommendation represents a recommendation for knowledge improvement
type KnowledgeRecommendation struct {
	Type        string             `json:"type"`        // "validation", "relationship", "update", "merge"
	Priority    string             `json:"priority"`    // "low", "medium", "high", "critical"
	Description string             `json:"description"` // Human-readable description
	ItemID      primitive.ObjectID `json:"item_id"`     // Primary item involved
	RelatedID   primitive.ObjectID `json:"related_id,omitempty"` // Related item (for relationships)
	Action      string             `json:"action"`      // Recommended action
	Confidence  float64            `json:"confidence"`  // Confidence in the recommendation
}

// KnowledgeAuditEntry represents an audit entry for knowledge operations
type KnowledgeAuditEntry struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	ItemID      primitive.ObjectID     `json:"item_id" bson:"item_id"`
	Action      string                 `json:"action" bson:"action"` // "create", "update", "delete", "validate", "relate"
	UserID      primitive.ObjectID     `json:"user_id" bson:"user_id"`
	Timestamp   int64                  `json:"timestamp" bson:"timestamp"`
	Changes     map[string]interface{} `json:"changes" bson:"changes"`
	Reason      string                 `json:"reason" bson:"reason"`
	IPAddress   string                 `json:"ip_address" bson:"ip_address"`
	UserAgent   string                 `json:"user_agent" bson:"user_agent"`
}

// KnowledgeExportFormat represents different export formats for knowledge
type KnowledgeExportFormat string

const (
	ExportFormatJSON     KnowledgeExportFormat = "json"
	ExportFormatXML      KnowledgeExportFormat = "xml"
	ExportFormatCSV      KnowledgeExportFormat = "csv"
	ExportFormatMarkdown KnowledgeExportFormat = "markdown"
	ExportFormatRDF      KnowledgeExportFormat = "rdf"
)

// KnowledgeExportOptions represents options for knowledge export
type KnowledgeExportOptions struct {
	Format           KnowledgeExportFormat `json:"format"`
	IncludeRelationships bool              `json:"include_relationships"`
	IncludeMetadata      bool              `json:"include_metadata"`
	IncludeUsageStats    bool              `json:"include_usage_stats"`
	FilterByType         []models.KnowledgeType `json:"filter_by_type,omitempty"`
	FilterByCategory     []string          `json:"filter_by_category,omitempty"`
	FilterByTags         []string          `json:"filter_by_tags,omitempty"`
	MinConfidence        float64           `json:"min_confidence"`
	ValidatedOnly        bool              `json:"validated_only"`
}

// KnowledgeImportResult represents the result of knowledge import
type KnowledgeImportResult struct {
	TotalItems      int      `json:"total_items"`
	ImportedItems   int      `json:"imported_items"`
	SkippedItems    int      `json:"skipped_items"`
	ErrorItems      int      `json:"error_items"`
	Errors          []string `json:"errors,omitempty"`
	ProcessingTime  int64    `json:"processing_time_ms"`
}

// KnowledgeVersionInfo represents version information for a knowledge item
type KnowledgeVersionInfo struct {
	ItemID      primitive.ObjectID `json:"item_id"`
	Version     int                `json:"version"`
	CreatedAt   int64              `json:"created_at"`
	CreatedBy   primitive.ObjectID `json:"created_by"`
	Changes     []string           `json:"changes"`
	ChangeType  string             `json:"change_type"` // "minor", "major", "critical"
}

// KnowledgeBackupInfo represents information about knowledge backups
type KnowledgeBackupInfo struct {
	BackupID    string `json:"backup_id"`
	CreatedAt   int64  `json:"created_at"`
	ItemCount   int64  `json:"item_count"`
	Size        int64  `json:"size_bytes"`
	Checksum    string `json:"checksum"`
	Description string `json:"description"`
}