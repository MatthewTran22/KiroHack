package knowledge

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-government-consultant/internal/models"
	"ai-government-consultant/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles knowledge management operations
type Service struct {
	repository RepositoryInterface
	logger     logger.Logger
}

// NewService creates a new knowledge management service
func NewService(db *mongo.Database, logger logger.Logger) *Service {
	return &Service{
		repository: NewRepository(db),
		logger:     logger,
	}
}

// GetRepository returns the repository instance
func (s *Service) GetRepository() RepositoryInterface {
	return s.repository
}

// CreateKnowledgeItem creates a new knowledge item
func (s *Service) CreateKnowledgeItem(ctx context.Context, item *models.KnowledgeItem) (*models.KnowledgeItem, error) {
	// Validate the item
	if err := item.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Set default values
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Version = 1
	item.IsActive = true

	// Initialize usage statistics
	item.Usage = models.KnowledgeUsage{
		AccessCount:        0,
		UsageContexts:      []string{},
		EffectivenessScore: 0.0,
	}

	// Initialize validation if not set
	if item.Validation.IsValidated == false && item.Validation.ValidatedBy == nil {
		item.Validation = models.KnowledgeValidation{
			IsValidated: false,
		}
	}

	// Extract keywords from content if not provided
	if len(item.Keywords) == 0 {
		item.Keywords = s.extractKeywords(item.Content)
	}

	// Create the item
	err := s.repository.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge item: %w", err)
	}

	s.logger.Info("Created knowledge item", map[string]interface{}{
		"id":       item.ID.Hex(),
		"type":     item.Type,
		"category": item.Category,
		"title":    item.Title,
	})

	return item, nil
}

// GetKnowledgeItem retrieves a knowledge item by ID
func (s *Service) GetKnowledgeItem(ctx context.Context, id primitive.ObjectID) (*models.KnowledgeItem, error) {
	item, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update usage statistics
	go s.updateUsageAsync(ctx, id, "retrieval")

	return item, nil
}

// UpdateKnowledgeItem updates an existing knowledge item
func (s *Service) UpdateKnowledgeItem(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) (*models.KnowledgeItem, error) {
	// Get the current item to validate updates
	_, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate critical fields if they're being updated
	if content, exists := updates["content"]; exists {
		if content == "" {
			return nil, fmt.Errorf("content cannot be empty")
		}
		// Re-extract keywords if content is updated
		if contentStr, ok := content.(string); ok {
			updates["keywords"] = s.extractKeywords(contentStr)
		}
	}

	if title, exists := updates["title"]; exists {
		if title == "" {
			return nil, fmt.Errorf("title cannot be empty")
		}
	}

	if confidence, exists := updates["confidence"]; exists {
		if conf, ok := confidence.(float64); ok {
			if conf < 0.0 || conf > 1.0 {
				return nil, fmt.Errorf("confidence must be between 0.0 and 1.0")
			}
		}
	}

	// Update the item
	err = s.repository.Update(ctx, id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update knowledge item: %w", err)
	}

	// Get the updated item
	updatedItem, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Updated knowledge item", map[string]interface{}{
		"id":      id.Hex(),
		"version": updatedItem.Version,
		"updates": len(updates),
	})

	return updatedItem, nil
}

// DeleteKnowledgeItem soft deletes a knowledge item
func (s *Service) DeleteKnowledgeItem(ctx context.Context, id primitive.ObjectID) error {
	err := s.repository.Delete(ctx, id)
	if err != nil {
		return err
	}

	s.logger.Info("Deleted knowledge item", map[string]interface{}{
		"id": id.Hex(),
	})

	return nil
}

// SearchKnowledge searches for knowledge items based on criteria
func (s *Service) SearchKnowledge(ctx context.Context, filter SearchFilter) ([]*models.KnowledgeItem, int64, error) {
	items, total, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Update usage statistics for accessed items
	go s.updateSearchUsageAsync(ctx, items, "search")

	s.logger.Debug("Knowledge search completed", map[string]interface{}{
		"query":         filter.Query,
		"results_count": len(items),
		"total_count":   total,
	})

	return items, total, nil
}

// GetRelatedKnowledge finds knowledge items related to a specific item
func (s *Service) GetRelatedKnowledge(ctx context.Context, itemID primitive.ObjectID, relationshipType models.RelationshipType, limit int) ([]*models.KnowledgeItem, error) {
	items, err := s.repository.GetRelatedItems(ctx, itemID, relationshipType, limit)
	if err != nil {
		return nil, err
	}

	// Update usage statistics
	go s.updateSearchUsageAsync(ctx, items, "relationship")

	return items, nil
}

// AddRelationship creates a relationship between two knowledge items
func (s *Service) AddRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType, strength float64, relationshipContext string) error {
	// Validate that both items exist
	_, err := s.repository.GetByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("source item not found: %w", err)
	}

	_, err = s.repository.GetByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("target item not found: %w", err)
	}

	// Validate strength
	if strength < 0.0 || strength > 1.0 {
		return fmt.Errorf("relationship strength must be between 0.0 and 1.0")
	}

	// Add the relationship
	err = s.repository.AddRelationship(ctx, sourceID, targetID, relType, strength, relationshipContext)
	if err != nil {
		return err
	}

	s.logger.Info("Added knowledge relationship", map[string]interface{}{
		"source_id": sourceID.Hex(),
		"target_id": targetID.Hex(),
		"type":      relType,
		"strength":  strength,
	})

	return nil
}

// RemoveRelationship removes a relationship between two knowledge items
func (s *Service) RemoveRelationship(ctx context.Context, sourceID, targetID primitive.ObjectID, relType models.RelationshipType) error {
	err := s.repository.RemoveRelationship(ctx, sourceID, targetID, relType)
	if err != nil {
		return err
	}

	s.logger.Info("Removed knowledge relationship", map[string]interface{}{
		"source_id": sourceID.Hex(),
		"target_id": targetID.Hex(),
		"type":      relType,
	})

	return nil
}

// ValidateKnowledgeItem validates a knowledge item
func (s *Service) ValidateKnowledgeItem(ctx context.Context, id primitive.ObjectID, validatedBy primitive.ObjectID, notes string, expiresAt *time.Time) error {
	updates := map[string]interface{}{
		"validation.is_validated":     true,
		"validation.validated_by":     validatedBy,
		"validation.validated_at":     time.Now(),
		"validation.validation_notes": notes,
	}

	if expiresAt != nil {
		updates["validation.expires_at"] = *expiresAt
	}

	_, err := s.UpdateKnowledgeItem(ctx, id, updates)
	if err != nil {
		return err
	}

	s.logger.Info("Validated knowledge item", map[string]interface{}{
		"id":           id.Hex(),
		"validated_by": validatedBy.Hex(),
		"expires_at":   expiresAt,
	})

	return nil
}

// InvalidateKnowledgeItem invalidates a knowledge item
func (s *Service) InvalidateKnowledgeItem(ctx context.Context, id primitive.ObjectID, reason string) error {
	updates := map[string]interface{}{
		"validation.is_validated":     false,
		"validation.validation_notes": reason,
		"validation.expires_at":       nil,
	}

	_, err := s.UpdateKnowledgeItem(ctx, id, updates)
	if err != nil {
		return err
	}

	s.logger.Info("Invalidated knowledge item", map[string]interface{}{
		"id":     id.Hex(),
		"reason": reason,
	})

	return nil
}

// ExtractKnowledgeFromDocument extracts knowledge items from a processed document
func (s *Service) ExtractKnowledgeFromDocument(ctx context.Context, document *models.Document, extractedBy primitive.ObjectID) ([]*models.KnowledgeItem, error) {
	var knowledgeItems []*models.KnowledgeItem

	// Extract different types of knowledge based on document content
	facts := s.extractFacts(document.Content)
	for _, fact := range facts {
		item := &models.KnowledgeItem{
			Content:  fact,
			Type:     models.KnowledgeTypeFact,
			Title:    s.generateTitle(fact),
			Category: string(document.Metadata.Category),
			Tags:     document.Metadata.Tags,
			Source: models.KnowledgeSource{
				Type:        "document",
				SourceID:    document.ID,
				Reference:   document.Name,
				Reliability: 0.8, // Default reliability for document-extracted knowledge
			},
			Confidence: 0.7, // Default confidence for extracted knowledge
			CreatedBy:  extractedBy,
		}

		createdItem, err := s.CreateKnowledgeItem(ctx, item)
		if err != nil {
			s.logger.Error("Failed to create knowledge item from document", err, map[string]interface{}{
				"document_id": document.ID.Hex(),
				"fact":        fact,
			})
			continue
		}

		knowledgeItems = append(knowledgeItems, createdItem)
	}

	// Extract rules and procedures
	rules := s.extractRules(document.Content)
	for _, rule := range rules {
		item := &models.KnowledgeItem{
			Content:  rule,
			Type:     models.KnowledgeTypeRule,
			Title:    s.generateTitle(rule),
			Category: string(document.Metadata.Category),
			Tags:     document.Metadata.Tags,
			Source: models.KnowledgeSource{
				Type:        "document",
				SourceID:    document.ID,
				Reference:   document.Name,
				Reliability: 0.9, // Higher reliability for rules
			},
			Confidence: 0.8,
			CreatedBy:  extractedBy,
		}

		createdItem, err := s.CreateKnowledgeItem(ctx, item)
		if err != nil {
			s.logger.Error("Failed to create rule knowledge item from document", err, map[string]interface{}{
				"document_id": document.ID.Hex(),
				"rule":        rule,
			})
			continue
		}

		knowledgeItems = append(knowledgeItems, createdItem)
	}

	// Extract procedures
	procedures := s.extractProcedures(document.Content)
	for _, procedure := range procedures {
		item := &models.KnowledgeItem{
			Content:  procedure,
			Type:     models.KnowledgeTypeProcedure,
			Title:    s.generateTitle(procedure),
			Category: string(document.Metadata.Category),
			Tags:     document.Metadata.Tags,
			Source: models.KnowledgeSource{
				Type:        "document",
				SourceID:    document.ID,
				Reference:   document.Name,
				Reliability: 0.85,
			},
			Confidence: 0.75,
			CreatedBy:  extractedBy,
		}

		createdItem, err := s.CreateKnowledgeItem(ctx, item)
		if err != nil {
			s.logger.Error("Failed to create procedure knowledge item from document", err, map[string]interface{}{
				"document_id": document.ID.Hex(),
				"procedure":   procedure,
			})
			continue
		}

		knowledgeItems = append(knowledgeItems, createdItem)
	}

	// Extract guidelines and best practices
	guidelines := s.extractGuidelines(document.Content)
	for _, guideline := range guidelines {
		item := &models.KnowledgeItem{
			Content:  guideline,
			Type:     models.KnowledgeTypeGuideline,
			Title:    s.generateTitle(guideline),
			Category: string(document.Metadata.Category),
			Tags:     document.Metadata.Tags,
			Source: models.KnowledgeSource{
				Type:        "document",
				SourceID:    document.ID,
				Reference:   document.Name,
				Reliability: 0.75,
			},
			Confidence: 0.7,
			CreatedBy:  extractedBy,
		}

		createdItem, err := s.CreateKnowledgeItem(ctx, item)
		if err != nil {
			s.logger.Error("Failed to create guideline knowledge item from document", err, map[string]interface{}{
				"document_id": document.ID.Hex(),
				"guideline":   guideline,
			})
			continue
		}

		knowledgeItems = append(knowledgeItems, createdItem)
	}

	// Automatically build relationships between extracted knowledge items
	if len(knowledgeItems) > 1 {
		err := s.buildExtractedKnowledgeRelationships(ctx, knowledgeItems)
		if err != nil {
			s.logger.Error("Failed to build relationships for extracted knowledge", err, map[string]interface{}{
				"document_id": document.ID.Hex(),
			})
		}
	}

	s.logger.Info("Extracted knowledge from document", map[string]interface{}{
		"document_id":     document.ID.Hex(),
		"knowledge_count": len(knowledgeItems),
		"facts":           len(facts),
		"rules":           len(rules),
		"procedures":      len(procedures),
		"guidelines":      len(guidelines),
	})

	return knowledgeItems, nil
}

// BuildKnowledgeGraph constructs relationships between knowledge items
func (s *Service) BuildKnowledgeGraph(ctx context.Context) error {
	// Get all active knowledge items
	filter := SearchFilter{
		Limit: 1000, // Process in batches
	}

	items, _, err := s.repository.Search(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get knowledge items for graph building: %w", err)
	}

	relationshipsCreated := 0

	// Build relationships based on content similarity and semantic connections
	for i, item1 := range items {
		for j, item2 := range items {
			if i >= j || item1.ID == item2.ID {
				continue // Skip self and already processed pairs
			}

			// Calculate relationship strength based on various factors
			strength := s.calculateRelationshipStrength(item1, item2)
			if strength < 0.3 { // Minimum threshold for relationships
				continue
			}

			// Determine relationship type
			relType := s.determineRelationshipType(item1, item2)

			// Check if relationship already exists
			exists := false
			for _, rel := range item1.Relationships {
				if rel.TargetID == item2.ID && rel.Type == relType {
					exists = true
					break
				}
			}

			if !exists {
				err := s.AddRelationship(ctx, item1.ID, item2.ID, relType, strength, "auto-generated")
				if err != nil {
					s.logger.Error("Failed to add auto-generated relationship", err, map[string]interface{}{
						"source_id": item1.ID.Hex(),
						"target_id": item2.ID.Hex(),
						"type":      relType,
					})
					continue
				}
				relationshipsCreated++
			}
		}
	}

	s.logger.Info("Built knowledge graph", map[string]interface{}{
		"items_processed":        len(items),
		"relationships_created":  relationshipsCreated,
	})

	return nil
}

// ValidateConsistency checks for conflicts and inconsistencies in the knowledge base
func (s *Service) ValidateConsistency(ctx context.Context) ([]ConsistencyIssue, error) {
	var issues []ConsistencyIssue

	// Get all active knowledge items
	filter := SearchFilter{
		Limit: 1000,
	}

	items, _, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge items for consistency validation: %w", err)
	}

	// Check for contradictory relationships
	for _, item := range items {
		contradictions := item.GetRelationshipsByType(models.RelationshipTypeContradicts)
		for _, contradiction := range contradictions {
			// Get the contradicting item
			contradictingItem, err := s.repository.GetByID(ctx, contradiction.TargetID)
			if err != nil {
				continue
			}

			issue := ConsistencyIssue{
				Type:        "contradiction",
				Description: fmt.Sprintf("Knowledge item '%s' contradicts '%s'", item.Title, contradictingItem.Title),
				ItemID1:     item.ID,
				ItemID2:     contradiction.TargetID,
				Severity:    "high",
				Confidence:  contradiction.Strength,
			}
			issues = append(issues, issue)
		}
	}

	// Check for expired knowledge items
	expiredItems, err := s.repository.GetExpiredItems(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired items: %w", err)
	}

	for _, item := range expiredItems {
		issue := ConsistencyIssue{
			Type:        "expired",
			Description: fmt.Sprintf("Knowledge item '%s' has expired", item.Title),
			ItemID1:     item.ID,
			Severity:    "medium",
			Confidence:  1.0,
		}
		issues = append(issues, issue)
	}

	// Check for low confidence items with high usage
	highUsageLowConfidenceItems := s.findHighUsageLowConfidenceItems(items)
	for _, item := range highUsageLowConfidenceItems {
		issue := ConsistencyIssue{
			Type:        "low_confidence_high_usage",
			Description: fmt.Sprintf("Knowledge item '%s' has low confidence (%.2f) but high usage (%d)", item.Title, item.Confidence, item.Usage.AccessCount),
			ItemID1:     item.ID,
			Severity:    "medium",
			Confidence:  1.0 - item.Confidence,
		}
		issues = append(issues, issue)
	}

	s.logger.Info("Validated knowledge consistency", map[string]interface{}{
		"items_checked": len(items),
		"issues_found":  len(issues),
	})

	return issues, nil
}

// ResolveConflict resolves a conflict between knowledge items
func (s *Service) ResolveConflict(ctx context.Context, item1ID, item2ID primitive.ObjectID, resolution ConflictResolution, resolvedBy primitive.ObjectID) error {
	switch resolution.Action {
	case "merge":
		return s.mergeKnowledgeItems(ctx, item1ID, item2ID, resolvedBy)
	case "supersede":
		return s.supersedeKnowledgeItem(ctx, resolution.PreferredItemID, resolution.SupersededItemID, resolvedBy)
	case "validate":
		return s.ValidateKnowledgeItem(ctx, resolution.PreferredItemID, resolvedBy, resolution.Notes, nil)
	case "invalidate":
		return s.InvalidateKnowledgeItem(ctx, resolution.SupersededItemID, resolution.Notes)
	default:
		return fmt.Errorf("unknown resolution action: %s", resolution.Action)
	}
}

// GetStatistics returns knowledge base statistics
func (s *Service) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	return s.repository.GetStatistics(ctx)
}

// CreateKnowledgeVersion creates a new version of a knowledge item
func (s *Service) CreateKnowledgeVersion(ctx context.Context, itemID primitive.ObjectID, updates map[string]interface{}, versionedBy primitive.ObjectID, changeType string, changes []string) (*models.KnowledgeItem, error) {
	// Get the current item
	currentItem, err := s.repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current item: %w", err)
	}

	// Create version info
	versionInfo := KnowledgeVersionInfo{
		ItemID:     itemID,
		Version:    currentItem.Version,
		CreatedAt:  time.Now().Unix(),
		CreatedBy:  versionedBy,
		Changes:    changes,
		ChangeType: changeType,
	}

	// Store version info in metadata
	if currentItem.Metadata == nil {
		currentItem.Metadata = make(map[string]interface{})
	}
	
	// Store version history
	versionHistory, exists := currentItem.Metadata["version_history"]
	if !exists {
		versionHistory = []KnowledgeVersionInfo{}
	}
	
	if versionHistorySlice, ok := versionHistory.([]KnowledgeVersionInfo); ok {
		versionHistorySlice = append(versionHistorySlice, versionInfo)
		updates["metadata.version_history"] = versionHistorySlice
	}

	// Update the item with new version
	updatedItem, err := s.UpdateKnowledgeItem(ctx, itemID, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to create new version: %w", err)
	}

	s.logger.Info("Created knowledge item version", map[string]interface{}{
		"item_id":     itemID.Hex(),
		"old_version": currentItem.Version,
		"new_version": updatedItem.Version,
		"change_type": changeType,
		"versioned_by": versionedBy.Hex(),
	})

	return updatedItem, nil
}

// GetKnowledgeVersionHistory retrieves the version history of a knowledge item
func (s *Service) GetKnowledgeVersionHistory(ctx context.Context, itemID primitive.ObjectID) ([]KnowledgeVersionInfo, error) {
	item, err := s.repository.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if item.Metadata == nil {
		return []KnowledgeVersionInfo{}, nil
	}

	versionHistory, exists := item.Metadata["version_history"]
	if !exists {
		return []KnowledgeVersionInfo{}, nil
	}

	if versionHistorySlice, ok := versionHistory.([]KnowledgeVersionInfo); ok {
		return versionHistorySlice, nil
	}

	return []KnowledgeVersionInfo{}, nil
}

// GetKnowledgeGraph constructs and returns the knowledge graph
func (s *Service) GetKnowledgeGraph(ctx context.Context, filter SearchFilter) (*KnowledgeGraph, error) {
	// Get knowledge items based on filter
	items, _, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge items for graph: %w", err)
	}

	// Build nodes
	nodes := make([]KnowledgeNode, len(items))
	for i, item := range items {
		nodes[i] = KnowledgeNode{
			ID:         item.ID.Hex(),
			Title:      item.Title,
			Type:       item.Type,
			Category:   item.Category,
			Confidence: item.Confidence,
			Usage:      item.Usage.AccessCount,
			Metadata: map[string]interface{}{
				"tags":     item.Tags,
				"keywords": item.Keywords,
				"version":  item.Version,
			},
		}
	}

	// Build edges from relationships
	var edges []KnowledgeEdge
	for _, item := range items {
		for _, rel := range item.Relationships {
			edges = append(edges, KnowledgeEdge{
				Source:   item.ID.Hex(),
				Target:   rel.TargetID.Hex(),
				Type:     rel.Type,
				Strength: rel.Strength,
				Context:  rel.Context,
			})
		}
	}

	graph := &KnowledgeGraph{
		Nodes: nodes,
		Edges: edges,
	}

	s.logger.Debug("Built knowledge graph", map[string]interface{}{
		"nodes": len(nodes),
		"edges": len(edges),
	})

	return graph, nil
}

// GetKnowledgeRecommendations provides recommendations for knowledge improvement
func (s *Service) GetKnowledgeRecommendations(ctx context.Context, limit int) ([]KnowledgeRecommendation, error) {
	var recommendations []KnowledgeRecommendation

	// Get expired items
	expiredItems, err := s.repository.GetExpiredItems(ctx, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired items: %w", err)
	}

	for _, item := range expiredItems {
		recommendations = append(recommendations, KnowledgeRecommendation{
			Type:        "validation",
			Priority:    "high",
			Description: fmt.Sprintf("Knowledge item '%s' has expired and needs revalidation", item.Title),
			ItemID:      item.ID,
			Action:      "revalidate",
			Confidence:  1.0,
		})
	}

	// Get high usage, low confidence items
	filter := SearchFilter{Limit: 100}
	items, _, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get items for recommendations: %w", err)
	}

	highUsageLowConfidence := s.findHighUsageLowConfidenceItems(items)
	for _, item := range highUsageLowConfidence {
		recommendations = append(recommendations, KnowledgeRecommendation{
			Type:        "update",
			Priority:    "medium",
			Description: fmt.Sprintf("Knowledge item '%s' has high usage but low confidence", item.Title),
			ItemID:      item.ID,
			Action:      "review_and_update",
			Confidence:  1.0 - item.Confidence,
		})
	}

	// Find items without relationships that could be related
	for _, item1 := range items {
		if len(item1.Relationships) == 0 {
			// Find potential relationships
			for _, item2 := range items {
				if item1.ID == item2.ID {
					continue
				}
				
				strength := s.calculateRelationshipStrength(item1, item2)
				if strength > 0.6 {
					recommendations = append(recommendations, KnowledgeRecommendation{
						Type:        "relationship",
						Priority:    "low",
						Description: fmt.Sprintf("Consider adding relationship between '%s' and '%s'", item1.Title, item2.Title),
						ItemID:      item1.ID,
						RelatedID:   item2.ID,
						Action:      "add_relationship",
						Confidence:  strength,
					})
				}
			}
		}
	}

	// Limit results
	if limit > 0 && len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// ExportKnowledge exports knowledge items in the specified format
func (s *Service) ExportKnowledge(ctx context.Context, options KnowledgeExportOptions) ([]byte, error) {
	// Build search filter from export options
	filter := SearchFilter{
		Type:          "",
		Category:      "",
		MinConfidence: options.MinConfidence,
		IsValidated:   &options.ValidatedOnly,
		Limit:         1000, // Export limit
	}

	if len(options.FilterByType) > 0 {
		// For simplicity, export first type only in this implementation
		filter.Type = options.FilterByType[0]
	}

	if len(options.FilterByCategory) > 0 {
		filter.Category = options.FilterByCategory[0]
	}

	if len(options.FilterByTags) > 0 {
		filter.Tags = options.FilterByTags
	}

	// Get knowledge items
	items, _, err := s.repository.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get items for export: %w", err)
	}

	// Export based on format
	switch options.Format {
	case ExportFormatJSON:
		return s.exportAsJSON(items, options)
	case ExportFormatCSV:
		return s.exportAsCSV(items, options)
	case ExportFormatMarkdown:
		return s.exportAsMarkdown(items, options)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", options.Format)
	}
}

// ImportKnowledge imports knowledge items from data
func (s *Service) ImportKnowledge(ctx context.Context, data []byte, format KnowledgeExportFormat, importedBy primitive.ObjectID) (*KnowledgeImportResult, error) {
	startTime := time.Now()
	result := &KnowledgeImportResult{
		ProcessingTime: 0,
	}

	var items []*models.KnowledgeItem
	var err error

	// Parse based on format
	switch format {
	case ExportFormatJSON:
		items, err = s.parseJSONImport(data)
	default:
		return nil, fmt.Errorf("unsupported import format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse import data: %w", err)
	}

	result.TotalItems = len(items)

	// Import each item
	for _, item := range items {
		item.CreatedBy = importedBy
		item.ID = primitive.NewObjectID() // Generate new ID

		_, err := s.CreateKnowledgeItem(ctx, item)
		if err != nil {
			result.ErrorItems++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to import item '%s': %v", item.Title, err))
			continue
		}

		result.ImportedItems++
	}

	result.ProcessingTime = time.Since(startTime).Milliseconds()

	s.logger.Info("Imported knowledge items", map[string]interface{}{
		"total":    result.TotalItems,
		"imported": result.ImportedItems,
		"errors":   result.ErrorItems,
		"format":   format,
	})

	return result, nil
}

// Helper methods

func (s *Service) updateUsageAsync(ctx context.Context, id primitive.ObjectID, context string) {
	err := s.repository.UpdateUsage(ctx, id, context)
	if err != nil {
		s.logger.Error("Failed to update usage statistics", err, map[string]interface{}{
			"id":      id.Hex(),
			"context": context,
		})
	}
}

func (s *Service) updateSearchUsageAsync(ctx context.Context, items []*models.KnowledgeItem, context string) {
	for _, item := range items {
		s.updateUsageAsync(ctx, item.ID, context)
	}
}

func (s *Service) extractKeywords(content string) []string {
	// Simple keyword extraction - in a real implementation, this would use NLP
	words := strings.Fields(strings.ToLower(content))
	keywordMap := make(map[string]bool)
	
	// Filter out common words and extract meaningful keywords
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
	}
	
	for _, word := range words {
		// Remove punctuation and check length
		word = strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(word) > 3 && !stopWords[word] {
			keywordMap[word] = true
		}
	}
	
	// Convert map to slice
	var keywords []string
	for keyword := range keywordMap {
		keywords = append(keywords, keyword)
		if len(keywords) >= 10 { // Limit to 10 keywords
			break
		}
	}
	
	return keywords
}

func (s *Service) extractFacts(content string) []string {
	// Simple fact extraction - look for sentences with factual indicators
	sentences := strings.Split(content, ".")
	var facts []string
	
	factIndicators := []string{
		"requires", "must", "shall", "is defined as", "means", "includes",
		"specifies", "establishes", "provides", "states", "mandates",
	}
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 20 { // Skip very short sentences
			continue
		}
		
		for _, indicator := range factIndicators {
			if strings.Contains(strings.ToLower(sentence), indicator) {
				facts = append(facts, sentence)
				break
			}
		}
		
		if len(facts) >= 5 { // Limit to 5 facts per document
			break
		}
	}
	
	return facts
}

func (s *Service) extractRules(content string) []string {
	// Simple rule extraction - look for sentences with rule indicators
	sentences := strings.Split(content, ".")
	var rules []string
	
	ruleIndicators := []string{
		"must", "shall", "should", "required to", "prohibited from",
		"not permitted", "forbidden", "mandatory", "obligated",
	}
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 20 { // Skip very short sentences
			continue
		}
		
		for _, indicator := range ruleIndicators {
			if strings.Contains(strings.ToLower(sentence), indicator) {
				rules = append(rules, sentence)
				break
			}
		}
		
		if len(rules) >= 3 { // Limit to 3 rules per document
			break
		}
	}
	
	return rules
}

func (s *Service) extractProcedures(content string) []string {
	// Extract procedures - look for step-by-step instructions
	sentences := strings.Split(content, ".")
	var procedures []string
	
	procedureIndicators := []string{
		"step", "first", "second", "third", "next", "then", "finally",
		"procedure", "process", "follow", "complete", "perform",
	}
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 25 { // Procedures tend to be longer
			continue
		}
		
		for _, indicator := range procedureIndicators {
			if strings.Contains(strings.ToLower(sentence), indicator) {
				procedures = append(procedures, sentence)
				break
			}
		}
		
		if len(procedures) >= 3 { // Limit to 3 procedures per document
			break
		}
	}
	
	return procedures
}

func (s *Service) extractGuidelines(content string) []string {
	// Extract guidelines and best practices
	sentences := strings.Split(content, ".")
	var guidelines []string
	
	guidelineIndicators := []string{
		"recommend", "suggest", "best practice", "guideline", "consider",
		"advisable", "preferred", "optimal", "effective", "should consider",
	}
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) < 20 {
			continue
		}
		
		for _, indicator := range guidelineIndicators {
			if strings.Contains(strings.ToLower(sentence), indicator) {
				guidelines = append(guidelines, sentence)
				break
			}
		}
		
		if len(guidelines) >= 3 { // Limit to 3 guidelines per document
			break
		}
	}
	
	return guidelines
}

func (s *Service) generateTitle(content string) string {
	// Generate a title from the first few words of content
	words := strings.Fields(content)
	if len(words) == 0 {
		return "Untitled Knowledge Item"
	}
	
	titleWords := words
	if len(words) > 8 {
		titleWords = words[:8]
	}
	
	title := strings.Join(titleWords, " ")
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	
	return title
}

func (s *Service) calculateRelationshipStrength(item1, item2 *models.KnowledgeItem) float64 {
	strength := 0.0
	
	// Same category increases strength
	if item1.Category == item2.Category {
		strength += 0.3
	}
	
	// Common tags increase strength
	commonTags := 0
	for _, tag1 := range item1.Tags {
		for _, tag2 := range item2.Tags {
			if tag1 == tag2 {
				commonTags++
				break
			}
		}
	}
	if len(item1.Tags) > 0 && len(item2.Tags) > 0 {
		strength += float64(commonTags) / float64(len(item1.Tags)+len(item2.Tags)) * 0.4
	}
	
	// Common keywords increase strength
	commonKeywords := 0
	for _, keyword1 := range item1.Keywords {
		for _, keyword2 := range item2.Keywords {
			if keyword1 == keyword2 {
				commonKeywords++
				break
			}
		}
	}
	if len(item1.Keywords) > 0 && len(item2.Keywords) > 0 {
		strength += float64(commonKeywords) / float64(len(item1.Keywords)+len(item2.Keywords)) * 0.3
	}
	
	return strength
}

func (s *Service) determineRelationshipType(item1, item2 *models.KnowledgeItem) models.RelationshipType {
	// Simple heuristics for relationship type determination
	
	// If both are rules and in same category, they might support each other
	if item1.Type == models.KnowledgeTypeRule && item2.Type == models.KnowledgeTypeRule {
		return models.RelationshipTypeSupports
	}
	
	// If one is a fact and another is a rule, the fact might support the rule
	if (item1.Type == models.KnowledgeTypeFact && item2.Type == models.KnowledgeTypeRule) ||
		(item1.Type == models.KnowledgeTypeRule && item2.Type == models.KnowledgeTypeFact) {
		return models.RelationshipTypeSupports
	}
	
	// Default to related_to
	return models.RelationshipTypeRelatedTo
}

func (s *Service) findHighUsageLowConfidenceItems(items []*models.KnowledgeItem) []*models.KnowledgeItem {
	var result []*models.KnowledgeItem
	
	for _, item := range items {
		// High usage (>10 accesses) but low confidence (<0.6)
		if item.Usage.AccessCount > 10 && item.Confidence < 0.6 {
			result = append(result, item)
		}
	}
	
	return result
}

func (s *Service) mergeKnowledgeItems(ctx context.Context, item1ID, item2ID primitive.ObjectID, mergedBy primitive.ObjectID) error {
	// Get both items
	item1, err := s.repository.GetByID(ctx, item1ID)
	if err != nil {
		return fmt.Errorf("failed to get first item: %w", err)
	}
	
	item2, err := s.repository.GetByID(ctx, item2ID)
	if err != nil {
		return fmt.Errorf("failed to get second item: %w", err)
	}
	
	// Create merged content
	mergedContent := item1.Content + "\n\n" + item2.Content
	mergedTitle := item1.Title + " (merged with " + item2.Title + ")"
	
	// Merge tags and keywords
	tagMap := make(map[string]bool)
	for _, tag := range item1.Tags {
		tagMap[tag] = true
	}
	for _, tag := range item2.Tags {
		tagMap[tag] = true
	}
	
	var mergedTags []string
	for tag := range tagMap {
		mergedTags = append(mergedTags, tag)
	}
	
	keywordMap := make(map[string]bool)
	for _, keyword := range item1.Keywords {
		keywordMap[keyword] = true
	}
	for _, keyword := range item2.Keywords {
		keywordMap[keyword] = true
	}
	
	var mergedKeywords []string
	for keyword := range keywordMap {
		mergedKeywords = append(mergedKeywords, keyword)
	}
	
	// Update the first item with merged content
	updates := map[string]interface{}{
		"content":            mergedContent,
		"title":              mergedTitle,
		"tags":               mergedTags,
		"keywords":           mergedKeywords,
		"confidence":         (item1.Confidence + item2.Confidence) / 2, // Average confidence
		"last_modified_by":   mergedBy,
	}
	
	_, err = s.UpdateKnowledgeItem(ctx, item1ID, updates)
	if err != nil {
		return fmt.Errorf("failed to update merged item: %w", err)
	}
	
	// Soft delete the second item
	err = s.DeleteKnowledgeItem(ctx, item2ID)
	if err != nil {
		return fmt.Errorf("failed to delete second item: %w", err)
	}
	
	s.logger.Info("Merged knowledge items", map[string]interface{}{
		"merged_item_id": item1ID.Hex(),
		"deleted_item_id": item2ID.Hex(),
		"merged_by": mergedBy.Hex(),
	})
	
	return nil
}

func (s *Service) supersedeKnowledgeItem(ctx context.Context, preferredID, supersededID primitive.ObjectID, supersededBy primitive.ObjectID) error {
	// Add supersedes relationship
	err := s.AddRelationship(ctx, preferredID, supersededID, models.RelationshipTypeSupersedes, 1.0, "conflict resolution")
	if err != nil {
		return fmt.Errorf("failed to add supersedes relationship: %w", err)
	}
	
	// Soft delete the superseded item
	err = s.DeleteKnowledgeItem(ctx, supersededID)
	if err != nil {
		return fmt.Errorf("failed to delete superseded item: %w", err)
	}
	
	s.logger.Info("Superseded knowledge item", map[string]interface{}{
		"preferred_id":   preferredID.Hex(),
		"superseded_id":  supersededID.Hex(),
		"superseded_by":  supersededBy.Hex(),
	})
	
	return nil
}

// buildExtractedKnowledgeRelationships builds relationships between knowledge items extracted from the same document
func (s *Service) buildExtractedKnowledgeRelationships(ctx context.Context, items []*models.KnowledgeItem) error {
	for i, item1 := range items {
		for j, item2 := range items {
			if i >= j {
				continue // Skip self and already processed pairs
			}

			// Determine relationship type based on knowledge types
			var relType models.RelationshipType
			var strength float64

			switch {
			case item1.Type == models.KnowledgeTypeRule && item2.Type == models.KnowledgeTypeProcedure:
				relType = models.RelationshipTypeImplements
				strength = 0.8
			case item1.Type == models.KnowledgeTypeProcedure && item2.Type == models.KnowledgeTypeRule:
				relType = models.RelationshipTypeImplements
				strength = 0.8
			case item1.Type == models.KnowledgeTypeFact && item2.Type == models.KnowledgeTypeRule:
				relType = models.RelationshipTypeSupports
				strength = 0.7
			case item1.Type == models.KnowledgeTypeRule && item2.Type == models.KnowledgeTypeFact:
				relType = models.RelationshipTypeSupports
				strength = 0.7
			case item1.Type == models.KnowledgeTypeGuideline && item2.Type == models.KnowledgeTypeBestPractice:
				relType = models.RelationshipTypeRelatedTo
				strength = 0.6
			case item1.Type == models.KnowledgeTypeBestPractice && item2.Type == models.KnowledgeTypeGuideline:
				relType = models.RelationshipTypeRelatedTo
				strength = 0.6
			default:
				// Same category items are related
				if item1.Category == item2.Category {
					relType = models.RelationshipTypeRelatedTo
					strength = 0.5
				} else {
					continue // Skip if no clear relationship
				}
			}

			// Add bidirectional relationship
			err := s.AddRelationship(ctx, item1.ID, item2.ID, relType, strength, "document extraction")
			if err != nil {
				s.logger.Error("Failed to add extracted knowledge relationship", err, map[string]interface{}{
					"source_id": item1.ID.Hex(),
					"target_id": item2.ID.Hex(),
					"type":      relType,
				})
			}
		}
	}

	return nil
}

// Helper methods for export/import

func (s *Service) exportAsJSON(items []*models.KnowledgeItem, options KnowledgeExportOptions) ([]byte, error) {
	exportData := make(map[string]interface{})
	exportData["items"] = items
	exportData["export_options"] = options
	exportData["exported_at"] = time.Now()
	exportData["total_items"] = len(items)

	return json.Marshal(exportData)
}

func (s *Service) exportAsCSV(items []*models.KnowledgeItem, options KnowledgeExportOptions) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Write header
	header := []string{"ID", "Title", "Type", "Category", "Content", "Confidence", "Created At", "Tags", "Keywords"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, item := range items {
		record := []string{
			item.ID.Hex(),
			item.Title,
			string(item.Type),
			item.Category,
			item.Content,
			fmt.Sprintf("%.2f", item.Confidence),
			item.CreatedAt.Format(time.RFC3339),
			strings.Join(item.Tags, ";"),
			strings.Join(item.Keywords, ";"),
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buffer.Bytes(), nil
}

func (s *Service) exportAsMarkdown(items []*models.KnowledgeItem, options KnowledgeExportOptions) ([]byte, error) {
	var buffer bytes.Buffer

	buffer.WriteString("# Knowledge Base Export\n\n")
	buffer.WriteString(fmt.Sprintf("Exported at: %s\n", time.Now().Format(time.RFC3339)))
	buffer.WriteString(fmt.Sprintf("Total items: %d\n\n", len(items)))

	// Group by category
	categoryMap := make(map[string][]*models.KnowledgeItem)
	for _, item := range items {
		categoryMap[item.Category] = append(categoryMap[item.Category], item)
	}

	for category, categoryItems := range categoryMap {
		buffer.WriteString(fmt.Sprintf("## %s\n\n", category))

		for _, item := range categoryItems {
			buffer.WriteString(fmt.Sprintf("### %s\n\n", item.Title))
			buffer.WriteString(fmt.Sprintf("**Type:** %s  \n", item.Type))
			buffer.WriteString(fmt.Sprintf("**Confidence:** %.2f  \n", item.Confidence))
			if len(item.Tags) > 0 {
				buffer.WriteString(fmt.Sprintf("**Tags:** %s  \n", strings.Join(item.Tags, ", ")))
			}
			buffer.WriteString(fmt.Sprintf("**Created:** %s  \n\n", item.CreatedAt.Format("2006-01-02")))
			buffer.WriteString(fmt.Sprintf("%s\n\n", item.Content))
			buffer.WriteString("---\n\n")
		}
	}

	return buffer.Bytes(), nil
}

func (s *Service) parseJSONImport(data []byte) ([]*models.KnowledgeItem, error) {
	var importData struct {
		Items []models.KnowledgeItem `json:"items"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert to pointers
	items := make([]*models.KnowledgeItem, len(importData.Items))
	for i := range importData.Items {
		items[i] = &importData.Items[i]
	}

	return items, nil
}