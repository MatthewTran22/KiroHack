package api

import (
	"net/http"
	"strconv"

	"ai-government-consultant/internal/document"
	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	service *document.Service
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(service *document.Service) *DocumentHandler {
	return &DocumentHandler{
		service: service,
	}
}

// UploadDocument handles document upload requests
// @Summary Upload a document
// @Description Upload a document for processing
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Param title formData string false "Document title"
// @Param author formData string false "Document author"
// @Param department formData string false "Department"
// @Param category formData string false "Document category"
// @Param tags formData string false "Comma-separated tags"
// @Success 200 {object} document.ProcessingResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/documents/upload [post]
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No file uploaded",
			Code:  "MISSING_FILE",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Invalid user ID format",
			Code:  "INVALID_USER_ID",
		})
		return
	}

	// Parse metadata from form
	metadata := models.DocumentMetadata{
		CustomFields: make(map[string]interface{}),
	}

	if title := c.PostForm("title"); title != "" {
		metadata.Title = &title
	}

	if author := c.PostForm("author"); author != "" {
		metadata.Author = &author
	}

	if department := c.PostForm("department"); department != "" {
		metadata.Department = &department
	}

	if category := c.PostForm("category"); category != "" {
		metadata.Category = models.DocumentCategory(category)
	}

	if tags := c.PostForm("tags"); tags != "" {
		// Parse comma-separated tags
		metadata.Tags = parseTags(tags)
	}

	if language := c.PostForm("language"); language != "" {
		metadata.Language = language
	} else {
		metadata.Language = "en" // Default to English
	}

	// Upload and process the document
	result, err := h.service.UploadDocument(file, metadata, userObjectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
			Code:  "UPLOAD_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDocument retrieves a document by ID
// @Summary Get document by ID
// @Description Retrieve a document by its ID
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} models.Document
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/documents/{id} [get]
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
		})
		return
	}

	doc, err := h.service.GetProcessingStatus(documentID)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Document not found",
				Code:  "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
			Code:  "GET_DOCUMENT_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// ProcessDocument processes a document by ID
// @Summary Process document
// @Description Process a document by its ID
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} models.Document
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/documents/{id}/process [post]
func (h *DocumentHandler) ProcessDocument(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
		})
		return
	}

	doc, err := h.service.ProcessDocument(documentID)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Document not found",
				Code:  "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
			Code:  "PROCESS_DOCUMENT_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// GetProcessingStatus gets the processing status of a document
// @Summary Get processing status
// @Description Get the processing status of a document
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} models.Document
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/documents/{id}/status [get]
func (h *DocumentHandler) GetProcessingStatus(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Document ID is required",
			Code:  "MISSING_DOCUMENT_ID",
		})
		return
	}

	doc, err := h.service.GetProcessingStatus(documentID)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Document not found",
				Code:  "DOCUMENT_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
			Code:  "GET_STATUS_FAILED",
		})
		return
	}

	// Return only status-related fields
	statusResponse := map[string]interface{}{
		"id":                   doc.ID,
		"name":                 doc.Name,
		"processing_status":    doc.ProcessingStatus,
		"uploaded_at":          doc.UploadedAt,
		"processing_timestamp": doc.ProcessingTimestamp,
		"processing_error":     doc.ProcessingError,
	}

	c.JSON(http.StatusOK, statusResponse)
}

// SearchDocuments searches for documents
// @Summary Search documents
// @Description Search for documents based on various criteria
// @Tags documents
// @Produce json
// @Param query query string false "Search query"
// @Param category query string false "Document category"
// @Param tags query string false "Comma-separated tags"
// @Param classification query string false "Security classification"
// @Param status query string false "Processing status"
// @Param date_from query string false "Date from (RFC3339 format)"
// @Param date_to query string false "Date to (RFC3339 format)"
// @Param limit query int false "Limit results" default(20)
// @Param skip query int false "Skip results" default(0)
// @Success 200 {object} SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/documents/search [get]
func (h *DocumentHandler) SearchDocuments(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Invalid user ID format",
			Code:  "INVALID_USER_ID",
		})
		return
	}

	// Parse search parameters
	filter := document.SearchFilter{
		Query:      c.Query("query"),
		UploadedBy: &userObjectID, // Only search user's own documents for security
	}

	if category := c.Query("category"); category != "" {
		filter.Category = models.DocumentCategory(category)
	}

	if tags := c.Query("tags"); tags != "" {
		filter.Tags = parseTags(tags)
	}

	if classification := c.Query("classification"); classification != "" {
		filter.Classification = classification
	}

	if status := c.Query("status"); status != "" {
		filter.Status = models.ProcessingStatus(status)
	}

	// Parse date filters
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if parsedDate, err := parseDate(dateFrom); err == nil {
			filter.DateFrom = &parsedDate
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		if parsedDate, err := parseDate(dateTo); err == nil {
			filter.DateTo = &parsedDate
		}
	}

	// Parse pagination
	if limit := c.Query("limit"); limit != "" {
		if parsedLimit, err := strconv.Atoi(limit); err == nil && parsedLimit > 0 {
			filter.Limit = parsedLimit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 20 // Default limit
	}

	if skip := c.Query("skip"); skip != "" {
		if parsedSkip, err := strconv.Atoi(skip); err == nil && parsedSkip >= 0 {
			filter.Skip = parsedSkip
		}
	}

	// Create repository and search
	repo := document.NewRepository(h.service.GetDatabase())
	documents, total, err := repo.Search(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
			Code:  "SEARCH_FAILED",
		})
		return
	}

	response := SearchResponse{
		Documents: documents,
		Total:     total,
		Limit:     filter.Limit,
		Skip:      filter.Skip,
	}

	c.JSON(http.StatusOK, response)
}

// ValidateDocument validates a document without uploading
// @Summary Validate document
// @Description Validate a document file without uploading it
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Success 200 {object} document.ValidationResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/documents/validate [post]
func (h *DocumentHandler) ValidateDocument(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No file uploaded",
			Code:  "MISSING_FILE",
		})
		return
	}

	// Validate the document
	result, err := h.service.ValidateDocument(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
			Code:  "VALIDATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RegisterRoutes registers document routes with the router
func (h *DocumentHandler) RegisterRoutes(router *gin.RouterGroup) {
	docs := router.Group("/documents")
	{
		docs.POST("/upload", h.UploadDocument)
		docs.POST("/validate", h.ValidateDocument)
		docs.GET("/search", h.SearchDocuments)
		docs.GET("/:id", h.GetDocument)
		docs.GET("/:id/status", h.GetProcessingStatus)
		docs.POST("/:id/process", h.ProcessDocument)
	}
}
