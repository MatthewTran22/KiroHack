package api

import (
	"net/http"
	"strconv"
	"strings"

	"ai-government-consultant/internal/consultation"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConsultationHandler handles consultation-related API endpoints
type ConsultationHandler struct {
	consultationService *consultation.Service
}

// NewConsultationHandler creates a new consultation handler
func NewConsultationHandler(consultationService *consultation.Service) *ConsultationHandler {
	return &ConsultationHandler{
		consultationService: consultationService,
	}
}

// CreateConsultationRequest represents a consultation creation request
type CreateConsultationRequest struct {
	Query               string                        `json:"query" binding:"required"`
	Type                models.ConsultationType       `json:"type" binding:"required"`
	Context             models.ConsultationContext    `json:"context,omitempty"`
	MaxSources          int                           `json:"max_sources,omitempty"`
	ConfidenceThreshold float64                       `json:"confidence_threshold,omitempty"`
	Tags                []string                      `json:"tags,omitempty"`
	IsMultiTurn         bool                          `json:"is_multi_turn,omitempty"`
}

// ContinueConsultationRequest represents a request to continue a multi-turn consultation
type ContinueConsultationRequest struct {
	Query               string                        `json:"query" binding:"required"`
	MaxSources          int                           `json:"max_sources,omitempty"`
	ConfidenceThreshold float64                       `json:"confidence_threshold,omitempty"`
}

// ConsultationSearchRequest represents a consultation search request
type ConsultationSearchRequest struct {
	Query     string                      `form:"query"`
	Type      models.ConsultationType     `form:"type"`
	UserID    string                      `form:"user_id"`
	Status    models.SessionStatus        `form:"status"`
	Tags      []string                    `form:"tags"`
	DateFrom  string                      `form:"date_from"`
	DateTo    string                      `form:"date_to"`
	Limit     int                         `form:"limit"`
	Skip      int                         `form:"skip"`
	SortBy    string                      `form:"sort_by"`
	SortOrder string                      `form:"sort_order"`
}

// CreateConsultation creates a new consultation session
func (h *ConsultationHandler) CreateConsultation(c *gin.Context) {
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
	if !user.HasPermission("consultations", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to create consultations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req CreateConsultationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Create consultation request
	consultationReq := &consultation.ConsultationRequest{
		Query:               req.Query,
		Type:                req.Type,
		UserID:              user.ID,
		Context:             req.Context,
		MaxSources:          req.MaxSources,
		ConfidenceThreshold: req.ConfidenceThreshold,
	}

	// Set defaults
	if consultationReq.MaxSources == 0 {
		consultationReq.MaxSources = 10
	}
	if consultationReq.ConfidenceThreshold == 0 {
		consultationReq.ConfidenceThreshold = 0.7
	}

	// Route to appropriate consultation method based on type
	var response *models.ConsultationResponse
	var err error

	switch req.Type {
	case models.ConsultationTypePolicy:
		response, err = h.consultationService.ConsultPolicy(c.Request.Context(), consultationReq)
	case models.ConsultationTypeStrategy:
		response, err = h.consultationService.ConsultStrategy(c.Request.Context(), consultationReq)
	case models.ConsultationTypeOperations:
		response, err = h.consultationService.ConsultOperations(c.Request.Context(), consultationReq)
	case models.ConsultationTypeTechnology:
		response, err = h.consultationService.ConsultTechnology(c.Request.Context(), consultationReq)
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid consultation type",
			Message: "Supported types: policy, strategy, operations, technology",
			Code:    "INVALID_CONSULTATION_TYPE",
		})
		return
	}

	if err != nil {
		if strings.Contains(err.Error(), "rate limit") {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "Rate limit exceeded",
				Message: err.Error(),
				Code:    "RATE_LIMIT_EXCEEDED",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Consultation failed",
			Message: err.Error(),
			Code:    "CONSULTATION_FAILED",
		})
		return
	}

	// Create consultation session (in a real implementation, this would be saved to database)
	session := &models.ConsultationSession{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		Type:      req.Type,
		Query:     req.Query,
		Response:  response,
		Context:   req.Context,
		Status:    models.SessionStatusCompleted,
		Tags:      req.Tags,
		IsMultiTurn: req.IsMultiTurn,
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Message: "Consultation completed successfully",
		Data: gin.H{
			"session_id": session.ID.Hex(),
			"session":    session,
		},
	})
}

// GetConsultation retrieves a consultation session by ID
func (h *ConsultationHandler) GetConsultation(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Session ID is required",
			Code:  "MISSING_SESSION_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid session ID format",
			Message: err.Error(),
			Code:    "INVALID_SESSION_ID",
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
	if !user.HasPermission("consultations", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read consultations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Note: In a real implementation, you would fetch the consultation session from the database
	// For now, we'll return a placeholder response
	_ = objID

	c.JSON(http.StatusOK, gin.H{
		"session": gin.H{
			"id":     sessionID,
			"status": "completed",
		},
	})
}

// ContinueConsultation continues a multi-turn consultation
func (h *ConsultationHandler) ContinueConsultation(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Session ID is required",
			Code:  "MISSING_SESSION_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid session ID format",
			Message: err.Error(),
			Code:    "INVALID_SESSION_ID",
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
	if !user.HasPermission("consultations", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to continue consultations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req ContinueConsultationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Note: In a real implementation, you would:
	// 1. Fetch the existing consultation session
	// 2. Verify it belongs to the user and is multi-turn
	// 3. Continue the consultation with the new query
	// 4. Update the session with the new turn
	// For now, we'll return a placeholder response

	_ = objID
	_ = req

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Consultation continued successfully",
		Data: gin.H{
			"session_id": sessionID,
			"turn_index": 2,
		},
	})
}

// ListConsultations returns a paginated list of consultation sessions
func (h *ConsultationHandler) ListConsultations(c *gin.Context) {
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
	if !user.HasPermission("consultations", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to list consultations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	skipStr := c.DefaultQuery("skip", "0")

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

	// Note: In a real implementation, you would fetch consultation sessions from the database
	// with proper filtering based on user permissions
	// For now, we'll return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"consultations": []gin.H{},
		"total":         0,
		"limit":         limit,
		"skip":          skip,
	})
}

// SearchConsultations searches consultation sessions based on criteria
func (h *ConsultationHandler) SearchConsultations(c *gin.Context) {
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
	if !user.HasPermission("consultations", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to search consultations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse search parameters
	var req ConsultationSearchRequest
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

	// Note: In a real implementation, you would perform the actual search
	// using the consultation service with the search parameters
	// For now, we'll return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"consultations": []gin.H{},
		"total":         0,
		"limit":         req.Limit,
		"skip":          req.Skip,
	})
}

// GetConsultationHistory returns the consultation history for a user
func (h *ConsultationHandler) GetConsultationHistory(c *gin.Context) {
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
	if !user.HasPermission("consultations", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read consultation history",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	skipStr := c.DefaultQuery("skip", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	skip, err := strconv.Atoi(skipStr)
	if err != nil || skip < 0 {
		skip = 0
	}

	// Note: In a real implementation, you would fetch the user's consultation history
	// from the database, ordered by creation date
	// For now, we'll return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"history": []gin.H{},
		"total":   0,
		"limit":   limit,
		"skip":    skip,
		"user_id": user.ID.Hex(),
	})
}

// DeleteConsultation deletes a consultation session
func (h *ConsultationHandler) DeleteConsultation(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Session ID is required",
			Code:  "MISSING_SESSION_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid session ID format",
			Message: err.Error(),
			Code:    "INVALID_SESSION_ID",
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
	if !user.HasPermission("consultations", "delete") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to delete consultations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Note: In a real implementation, you would:
	// 1. Fetch the consultation session from the database
	// 2. Verify it belongs to the user or user has admin permissions
	// 3. Delete the session from the database
	// For now, we'll return a success response

	_ = objID

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Consultation deleted successfully",
	})
}

// GetRecommendation retrieves a specific recommendation from a consultation
func (h *ConsultationHandler) GetRecommendation(c *gin.Context) {
	sessionID := c.Param("session_id")
	recommendationID := c.Param("recommendation_id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Session ID is required",
			Code:  "MISSING_SESSION_ID",
		})
		return
	}

	if recommendationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Recommendation ID is required",
			Code:  "MISSING_RECOMMENDATION_ID",
		})
		return
	}

	// Validate ObjectIDs
	sessionObjID, err := primitive.ObjectIDFromHex(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid session ID format",
			Message: err.Error(),
			Code:    "INVALID_SESSION_ID",
		})
		return
	}

	recommendationObjID, err := primitive.ObjectIDFromHex(recommendationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid recommendation ID format",
			Message: err.Error(),
			Code:    "INVALID_RECOMMENDATION_ID",
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
	if !user.HasPermission("consultations", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read recommendations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Note: In a real implementation, you would:
	// 1. Fetch the consultation session from the database
	// 2. Verify it belongs to the user or user has appropriate permissions
	// 3. Find the specific recommendation within the session
	// For now, we'll return a placeholder response

	_ = sessionObjID
	_ = recommendationObjID

	c.JSON(http.StatusOK, gin.H{
		"recommendation": gin.H{
			"id":          recommendationID,
			"session_id":  sessionID,
			"title":       "Sample Recommendation",
			"description": "This is a sample recommendation",
		},
	})
}

// ExplainRecommendation provides detailed explanation for a recommendation
func (h *ConsultationHandler) ExplainRecommendation(c *gin.Context) {
	sessionID := c.Param("session_id")
	recommendationID := c.Param("recommendation_id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Session ID is required",
			Code:  "MISSING_SESSION_ID",
		})
		return
	}

	if recommendationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Recommendation ID is required",
			Code:  "MISSING_RECOMMENDATION_ID",
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
	if !user.HasPermission("consultations", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read recommendation explanations",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Note: In a real implementation, you would use the consultation service
	// to generate a detailed explanation for the recommendation
	// For now, we'll return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"explanation": gin.H{
			"recommendation_id": recommendationID,
			"session_id":        sessionID,
			"reasoning":         "This recommendation is based on analysis of relevant documents and best practices.",
			"sources":           []gin.H{},
			"confidence_factors": []string{
				"Strong alignment with policy objectives",
				"Supported by historical precedents",
				"Low implementation risk",
			},
		},
	})
}

// GetConsultationAnalytics returns analytics for consultation usage
func (h *ConsultationHandler) GetConsultationAnalytics(c *gin.Context) {
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

	// Check permissions (admin only for system-wide analytics)
	if !user.IsAdmin() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view analytics",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse time range parameters
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	// Note: In a real implementation, you would generate analytics
	// based on consultation data from the database
	// For now, we'll return placeholder analytics

	_ = dateFrom
	_ = dateTo

	c.JSON(http.StatusOK, gin.H{
		"analytics": gin.H{
			"total_consultations": 0,
			"consultations_by_type": gin.H{
				"policy":     0,
				"strategy":   0,
				"operations": 0,
				"technology": 0,
			},
			"average_response_time": "0s",
			"user_satisfaction":     0.0,
			"most_common_topics":    []string{},
		},
	})
}