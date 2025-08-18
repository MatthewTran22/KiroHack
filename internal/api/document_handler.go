package api

import (
	"net/http"
	"strconv"
	"strings"

	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentHandler handles document-related API endpoints
type DocumentHandler struct {
	documentService *document.Service
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(documentService *document.Service) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
	}
}

// UploadDocumentRequest represents metadata for document upload
type UploadDocumentRequest struct {
	Title      string                   `form:"title"`
	Author     string                   `form:"author"`
	Department string                   `form:"department"`
	Category   models.DocumentCategory  `form:"category" binding:"required"`
	Tags       string                   `form:"tags"` // Comma-separated
	Language   string                   `form:"language"`
}

// DocumentSearchRequest represents a document search request
type DocumentSearchRequest struct {
	Query      string                   `form:"query"`
	Category   models.DocumentCategory  `form:"category"`
	Tags       []string                 `form:"tags"`
	Department string                   `form:"department"`
	Author     string                   `form:"author"`
	Limit      int                      `form:"limit"`
	Skip       int                      `form:"skip"`
	SortBy     string                   `form:"sort_by"`
	SortOrder  string                   `form:"sort_order"`
}

// UploadDocument handles document upload
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
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
	if !user.HasPermission("documents", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to upload documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse multipart form
	var req UploadDocumentRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "File is required",
			Message: err.Error(),
			Code:    "MISSING_FILE",
		})
		return
	}

	// Create document metadata
	metadata := models.DocumentMetadata{
		Category: req.Category,
		Tags:     parseTags(req.Tags),
		Language: req.Language,
	}

	if req.Title != "" {
		metadata.Title = &req.Title
	}
	if req.Author != "" {
		metadata.Author = &req.Author
	}
	if req.Department != "" {
		metadata.Department = &req.Department
	}

	// Upload document
	result, err := h.documentService.UploadDocument(file, metadata, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Document upload failed",
			Message: err.Error(),
			Code:    "UPLOAD_FAILED",
		})
		return
	}

	if result.Status == "failed" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Document validation failed",
			Message: result.Message,
			Code:    "VALIDATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Message: result.Message,
		Data: gin.H{
			"document_id": result.DocumentID.Hex(),
			"status":      result.Status,
		},
	})
}

// GetDocument retrieves a document by ID
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
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
	if !user.HasPermission("documents", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get document processing status
	doc, err := h.documentService.GetProcessingStatus(documentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Document not found",
				Message: err.Error(),
				Code:    "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve document",
			Message: err.Error(),
			Code:    "RETRIEVAL_FAILED",
		})
		return
	}

	// Check if user can access this classification level
	if !user.CanAccessClassification(doc.Classification.Level) {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient security clearance",
			Code:  "INSUFFICIENT_CLEARANCE",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"document": doc,
	})
}

// ProcessDocument triggers document processing
func (h *DocumentHandler) ProcessDocument(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
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
	if !user.HasPermission("documents", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to process documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Process document
	doc, err := h.documentService.ProcessDocument(documentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Document not found",
				Message: err.Error(),
				Code:    "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Document processing failed",
			Message: err.Error(),
			Code:    "PROCESSING_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Document processed successfully",
		Data: gin.H{
			"document": doc,
		},
	})
}

// GetProcessingStatus returns the processing status of a document
func (h *DocumentHandler) GetProcessingStatus(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
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
	if !user.HasPermission("documents", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get processing status
	doc, err := h.documentService.GetProcessingStatus(documentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Document not found",
				Message: err.Error(),
				Code:    "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get processing status",
			Message: err.Error(),
			Code:    "STATUS_RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"document_id":        doc.ID.Hex(),
		"processing_status":  doc.ProcessingStatus,
		"processing_error":   doc.ProcessingError,
		"processing_timestamp": doc.ProcessingTimestamp,
	})
}

// SearchDocuments searches for documents based on criteria
func (h *DocumentHandler) SearchDocuments(c *gin.Context) {
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
	if !user.HasPermission("documents", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to search documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse search parameters
	var req DocumentSearchRequest
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
	// using the document service with the search parameters
	// For now, we'll return a placeholder response

	c.JSON(http.StatusOK, SearchResponse{
		Documents: []*models.Document{},
		Total:     0,
		Limit:     req.Limit,
		Skip:      req.Skip,
	})
}

// ListDocuments returns a paginated list of documents
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
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
	if !user.HasPermission("documents", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to list documents",
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

	// Note: In a real implementation, you would fetch documents from the database
	// with proper filtering based on user's security clearance
	// For now, we'll return a placeholder response

	c.JSON(http.StatusOK, SearchResponse{
		Documents: []*models.Document{},
		Total:     0,
		Limit:     limit,
		Skip:      skip,
	})
}

// UpdateDocument updates document metadata
func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
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
	if !user.HasPermission("documents", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to update documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req struct {
		Title      *string                  `json:"title,omitempty"`
		Author     *string                  `json:"author,omitempty"`
		Department *string                  `json:"department,omitempty"`
		Category   *models.DocumentCategory `json:"category,omitempty"`
		Tags       []string                 `json:"tags,omitempty"`
		Language   *string                  `json:"language,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Note: In a real implementation, you would update the document in the database
	// For now, we'll return a success response

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Document updated successfully",
		Data: gin.H{
			"document_id": documentID,
		},
	})
}

// DeleteDocument deletes a document
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
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
	if !user.HasPermission("documents", "delete") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to delete documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Validate ObjectID
	_, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid document ID format",
			Message: err.Error(),
			Code:    "INVALID_DOCUMENT_ID",
		})
		return
	}

	// Note: In a real implementation, you would delete the document from the database
	// and any associated files from storage
	// For now, we'll return a success response

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Document deleted successfully",
	})
}

// ValidateDocument validates a document file before upload
func (h *DocumentHandler) ValidateDocument(c *gin.Context) {
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
	if !user.HasPermission("documents", "write") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to validate documents",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "File is required",
			Message: err.Error(),
			Code:    "MISSING_FILE",
		})
		return
	}

	// Validate document
	validation, err := h.documentService.ValidateDocument(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Document validation failed",
			Message: err.Error(),
			Code:    "VALIDATION_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  validation.Valid,
		"errors": validation.Errors,
		"format": validation.Format,
		"size":   validation.Size,
	})
}

// GetDocumentContent returns the processed content of a document
func (h *DocumentHandler) GetDocumentContent(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
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
	if !user.HasPermission("documents", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to read document content",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get document
	doc, err := h.documentService.GetProcessingStatus(documentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Document not found",
				Message: err.Error(),
				Code:    "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve document",
			Message: err.Error(),
			Code:    "RETRIEVAL_FAILED",
		})
		return
	}

	// Check if user can access this classification level
	if !user.CanAccessClassification(doc.Classification.Level) {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient security clearance",
			Code:  "INSUFFICIENT_CLEARANCE",
		})
		return
	}

	// Check if document is processed
	if !doc.IsProcessed() {
		c.JSON(http.StatusAccepted, gin.H{
			"message":           "Document is still being processed",
			"processing_status": doc.ProcessingStatus,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"document_id":        doc.ID.Hex(),
		"content":            doc.Content,
		"extracted_entities": doc.ExtractedEntities,
		"processing_timestamp": doc.ProcessingTimestamp,
	})
}