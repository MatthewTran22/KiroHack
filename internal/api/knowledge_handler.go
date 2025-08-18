package api

import (
	"net/http"
	"strconv"
	"strings"

	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KnowledgeHandler handles knowledge management API endpoints
type KnowledgeHandler struct {
	knowledgeService KnowledgeServiceInterface
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(knowledgeService KnowledgeServiceInterface) *KnowledgeHandler {
	return &KnowledgeHandler{
		knowledgeService: knowledgeService,
	}
}

// CreateKnowledgeRequest represents a knowledge item creation request
type CreateKnowledgeRequest struct {
	Title       string                    `json:"title" binding:"required"`
	Content     string                    `json:"content" binding:"required"`
	Type        KnowledgeType             `json:"type" binding:"required"`
	Category    string                    `json:"category,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	Source      *DocumentReference        `json:"source,omitempty"`
	Metadata    map[string]interface{}    `json:"metadata,omitempty"`
}

// UpdateKnowledgeRequest represents a knowledge item update request
type UpdateKnowledgeRequest struct {
	Title    *string                `json:"title,omitempty"`
	Content  *string                `json:"content,omitempty"`
	Category *string                `json:"category,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// KnowledgeSearchRequest represents a knowledge search request
type KnowledgeSearchRequest struct {
	Query     string        `form:"query"`
	Type      KnowledgeType `form:"type"`
	Category  string        `form:"category"`
	Tags      []string      `form:"tags"`
	Limit     int           `form:"limit"`
	Skip      int           `form:"skip"`
	SortBy    string        `form:"sort_by"`
	SortOrder string        `form:"sort_order"`
	Threshold float64       `form:"threshold"`
}

// CreateKnowledge creates a new knowledge item
func (h *KnowledgeHandler) CreateKnowledge(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to create knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req CreateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Create knowledge item
	knowledgeItem := &KnowledgeItem{
		ID:       primitive.NewObjectID(),
		Title:    req.Title,
		Content:  req.Content,
		Type:     req.Type,
		Category: req.Category,
		Tags:     req.Tags,
		Source:   req.Source,
		Metadata: req.Metadata,
		CreatedBy: user.ID,
	}

	// Add knowledge item using service
	err := h.knowledgeService.AddKnowledge(c.Request.Context(), knowledgeItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create knowledge item",
			Message: err.Error(),
			Code:    "CREATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "Knowledge item created successfully",
		Data: gin.H{
			"knowledge_id": knowledgeItem.ID.Hex(),
			"knowledge":    knowledgeItem,
		},
	})
}

// GetKnowledge retrieves a knowledge item by ID
func (h *KnowledgeHandler) GetKnowledge(c *gin.Context) {
	knowledgeID := c.Param("id")
	if knowledgeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Knowledge ID is required",
			Code:  "MISSING_KNOWLEDGE_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(knowledgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid knowledge ID format",
			Message: err.Error(),
			Code:    "INVALID_KNOWLEDGE_ID",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get knowledge item
	knowledgeItem, err := h.knowledgeService.GetKnowledge(c.Request.Context(), objID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Knowledge item not found",
				Message: err.Error(),
				Code:    "KNOWLEDGE_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve knowledge item",
			Message: err.Error(),
			Code:    "RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"knowledge": knowledgeItem,
	})
}

// UpdateKnowledge updates a knowledge item
func (h *KnowledgeHandler) UpdateKnowledge(c *gin.Context) {
	knowledgeID := c.Param("id")
	if knowledgeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Knowledge ID is required",
			Code:  "MISSING_KNOWLEDGE_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(knowledgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid knowledge ID format",
			Message: err.Error(),
			Code:    "INVALID_KNOWLEDGE_ID",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to update knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Prepare updates map
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	// Update knowledge item
	err = h.knowledgeService.UpdateKnowledge(c.Request.Context(), objID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Knowledge item not found",
				Message: err.Error(),
				Code:    "KNOWLEDGE_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update knowledge item",
			Message: err.Error(),
			Code:    "UPDATE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Knowledge item updated successfully",
		Data: gin.H{
			"knowledge_id": knowledgeID,
		},
	})
}

// DeleteKnowledge deletes a knowledge item
func (h *KnowledgeHandler) DeleteKnowledge(c *gin.Context) {
	knowledgeID := c.Param("id")
	if knowledgeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Knowledge ID is required",
			Code:  "MISSING_KNOWLEDGE_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(knowledgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid knowledge ID format",
			Message: err.Error(),
			Code:    "INVALID_KNOWLEDGE_ID",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "delete") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to delete knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Delete knowledge item
	err = h.knowledgeService.DeleteKnowledge(c.Request.Context(), objID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Knowledge item not found",
				Message: err.Error(),
				Code:    "KNOWLEDGE_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete knowledge item",
			Message: err.Error(),
			Code:    "DELETE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Knowledge item deleted successfully",
	})
}

// SearchKnowledge searches for knowledge items
func (h *KnowledgeHandler) SearchKnowledge(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to search knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse search parameters
	var req KnowledgeSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid search parameters",
			Message: err.Error(),
			Code:    "INVALID_SEARCH_PARAMS",
		})
		return
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Skip < 0 {
		req.Skip = 0
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.7
	}

	// Create search filter
	filter := &KnowledgeFilter{
		Query:     req.Query,
		Type:      req.Type,
		Category:  req.Category,
		Tags:      req.Tags,
		Limit:     req.Limit,
		Skip:      req.Skip,
		Threshold: req.Threshold,
	}

	// Search knowledge items
	results, err := h.knowledgeService.SearchKnowledge(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Knowledge search failed",
			Message: err.Error(),
			Code:    "SEARCH_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
		"limit":   req.Limit,
		"skip":    req.Skip,
	})
}

// ListKnowledge returns a paginated list of knowledge items
func (h *KnowledgeHandler) ListKnowledge(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to list knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	skipStr := c.DefaultQuery("skip", "0")
	category := c.Query("category")
	knowledgeType := c.Query("type")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	skip, err := strconv.Atoi(skipStr)
	if err != nil || skip < 0 {
		skip = 0
	}

	// Create filter for listing
	filter := &KnowledgeFilter{
		Category: category,
		Limit:    limit,
		Skip:     skip,
	}

	if knowledgeType != "" {
		filter.Type = KnowledgeType(knowledgeType)
	}

	// List knowledge items
	results, err := h.knowledgeService.ListKnowledge(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list knowledge items",
			Message: err.Error(),
			Code:    "LIST_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"knowledge_items": results,
		"total":           len(results),
		"limit":           limit,
		"skip":            skip,
	})
}

// GetRelatedKnowledge returns knowledge items related to a specific item
func (h *KnowledgeHandler) GetRelatedKnowledge(c *gin.Context) {
	knowledgeID := c.Param("id")
	if knowledgeID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Knowledge ID is required",
			Code:  "MISSING_KNOWLEDGE_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(knowledgeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid knowledge ID format",
			Message: err.Error(),
			Code:    "INVALID_KNOWLEDGE_ID",
		})
		return
	}

	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read related knowledge items",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get related knowledge items
	relatedItems, err := h.knowledgeService.GetRelatedKnowledge(c.Request.Context(), objID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get related knowledge items",
			Message: err.Error(),
			Code:    "RELATED_RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"knowledge_id":    knowledgeID,
		"related_items":   relatedItems,
		"total_related":   len(relatedItems),
	})
}

// GetKnowledgeGraph returns the knowledge graph structure
func (h *KnowledgeHandler) GetKnowledgeGraph(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view knowledge graph",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse optional parameters
	category := c.Query("category")
	depth := c.DefaultQuery("depth", "2")
	maxNodes := c.DefaultQuery("max_nodes", "100")

	depthInt, err := strconv.Atoi(depth)
	if err != nil || depthInt < 1 {
		depthInt = 2
	}
	if depthInt > 5 {
		depthInt = 5
	}

	maxNodesInt, err := strconv.Atoi(maxNodes)
	if err != nil || maxNodesInt < 1 {
		maxNodesInt = 100
	}
	if maxNodesInt > 500 {
		maxNodesInt = 500
	}

	// Build knowledge graph
	graph, err := h.knowledgeService.BuildKnowledgeGraph(c.Request.Context(), &GraphOptions{
		Category: category,
		Depth:    depthInt,
		MaxNodes: maxNodesInt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to build knowledge graph",
			Message: err.Error(),
			Code:    "GRAPH_BUILD_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"graph": graph,
	})
}

// GetKnowledgeCategories returns available knowledge categories
func (h *KnowledgeHandler) GetKnowledgeCategories(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read knowledge categories",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get categories
	categories, err := h.knowledgeService.GetCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get knowledge categories",
			Message: err.Error(),
			Code:    "CATEGORIES_RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"total":      len(categories),
	})
}

// GetKnowledgeTypes returns available knowledge types
func (h *KnowledgeHandler) GetKnowledgeTypes(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions
	if !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read knowledge types",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Return available knowledge types
	types := []gin.H{
		{"value": "fact", "label": "Fact", "description": "Factual information and data"},
		{"value": "procedure", "label": "Procedure", "description": "Step-by-step procedures and processes"},
		{"value": "policy", "label": "Policy", "description": "Policy information and guidelines"},
		{"value": "regulation", "label": "Regulation", "description": "Regulatory requirements and compliance"},
		{"value": "best_practice", "label": "Best Practice", "description": "Proven best practices and recommendations"},
		{"value": "case_study", "label": "Case Study", "description": "Real-world examples and case studies"},
		{"value": "reference", "label": "Reference", "description": "Reference materials and documentation"},
	}

	c.JSON(http.StatusOK, gin.H{
		"types": types,
		"total": len(types),
	})
}

// GetKnowledgeStats returns statistics about the knowledge base
func (h *KnowledgeHandler) GetKnowledgeStats(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions (admin only for detailed stats)
	if !user.IsAdmin() && !user.HasPermission("knowledge", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view knowledge statistics",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get knowledge statistics
	stats, err := h.knowledgeService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get knowledge statistics",
			Message: err.Error(),
			Code:    "STATS_RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
	})
}