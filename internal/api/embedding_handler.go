package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"ai-government-consultant/internal/embedding"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmbeddingService defines the interface for embedding operations
type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
	GenerateDocumentEmbedding(ctx context.Context, documentID primitive.ObjectID) error
	GenerateKnowledgeEmbedding(ctx context.Context, knowledgeID primitive.ObjectID) error
	VectorSearch(ctx context.Context, query string, options *embedding.SearchOptions) ([]embedding.SearchResult, error)
	GetSimilarDocuments(ctx context.Context, documentID primitive.ObjectID, limit int) ([]embedding.SearchResult, error)
	GetSimilarKnowledge(ctx context.Context, knowledgeID primitive.ObjectID, limit int) ([]embedding.SearchResult, error)
	ClearCache(ctx context.Context) error
}

// EmbeddingRepository defines the interface for embedding repository operations
type EmbeddingRepository interface {
	GetEmbeddingStats(ctx context.Context) (*embedding.EmbeddingStats, error)
}

// EmbeddingPipeline defines the interface for embedding pipeline operations
type EmbeddingPipeline interface {
	ProcessAllDocuments(ctx context.Context) (*embedding.ProcessResult, error)
	ProcessAllKnowledgeItems(ctx context.Context) (*embedding.ProcessResult, error)
	ProcessSpecificDocuments(ctx context.Context, documentIDs []primitive.ObjectID) (*embedding.ProcessResult, error)
	ProcessSpecificKnowledgeItems(ctx context.Context, knowledgeIDs []primitive.ObjectID) (*embedding.ProcessResult, error)
}

// EmbeddingHandler handles embedding-related HTTP requests
type EmbeddingHandler struct {
	service    EmbeddingService
	repository EmbeddingRepository
	pipeline   EmbeddingPipeline
}

// NewEmbeddingHandler creates a new embedding handler
func NewEmbeddingHandler(service EmbeddingService, repository EmbeddingRepository, pipeline EmbeddingPipeline) *EmbeddingHandler {
	return &EmbeddingHandler{
		service:    service,
		repository: repository,
		pipeline:   pipeline,
	}
}

// GenerateEmbeddingRequest represents the request to generate an embedding
type GenerateEmbeddingRequest struct {
	Text string `json:"text" binding:"required"`
}

// GenerateEmbeddingResponse represents the response from generating an embedding
type GenerateEmbeddingResponse struct {
	Text       string    `json:"text"`
	Embeddings []float64 `json:"embeddings"`
	Dimensions int       `json:"dimensions"`
}

// VectorSearchRequest represents a vector search request
type VectorSearchRequest struct {
	Query      string                 `json:"query" binding:"required"`
	Limit      int                    `json:"limit,omitempty"`
	Threshold  float64                `json:"threshold,omitempty"`
	Collection string                 `json:"collection,omitempty"`
	Filters    map[string]interface{} `json:"filters,omitempty"`
}

// VectorSearchResponse represents a vector search response
type VectorSearchResponse struct {
	Query   string                   `json:"query"`
	Results []embedding.SearchResult `json:"results"`
	Count   int                      `json:"count"`
}

// ProcessEmbeddingsRequest represents a request to process embeddings
type ProcessEmbeddingsRequest struct {
	DocumentIDs  []string `json:"document_ids,omitempty"`
	KnowledgeIDs []string `json:"knowledge_ids,omitempty"`
	ProcessAll   bool     `json:"process_all,omitempty"`
}

// ProcessEmbeddingsResponse represents the response from processing embeddings
type ProcessEmbeddingsResponse struct {
	TotalProcessed int      `json:"total_processed"`
	Successful     int      `json:"successful"`
	Failed         int      `json:"failed"`
	Duration       string   `json:"duration"`
	Errors         []string `json:"errors,omitempty"`
}

// RegisterEmbeddingRoutes registers embedding routes with the router
func (h *EmbeddingHandler) RegisterRoutes(router *gin.RouterGroup) {
	embeddings := router.Group("/embeddings")
	{
		embeddings.POST("/generate", h.GenerateEmbedding)
		embeddings.POST("/search", h.VectorSearch)
		embeddings.POST("/process", h.ProcessEmbeddings)
		embeddings.GET("/stats", h.GetEmbeddingStats)
		embeddings.POST("/documents/:id/embedding", h.GenerateDocumentEmbedding)
		embeddings.POST("/knowledge/:id/embedding", h.GenerateKnowledgeEmbedding)
		embeddings.GET("/documents/:id/similar", h.GetSimilarDocuments)
		embeddings.GET("/knowledge/:id/similar", h.GetSimilarKnowledge)
		embeddings.DELETE("/cache", h.ClearCache)
	}
}

// GenerateEmbedding generates an embedding for the provided text
// @Summary Generate text embedding
// @Description Generate vector embedding for the provided text using Gemini API
// @Tags embeddings
// @Accept json
// @Produce json
// @Param request body GenerateEmbeddingRequest true "Text to generate embedding for"
// @Success 200 {object} GenerateEmbeddingResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/generate [post]
func (h *EmbeddingHandler) GenerateEmbedding(c *gin.Context) {
	var req GenerateEmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	embeddings, err := h.service.GenerateEmbedding(c.Request.Context(), req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate embedding",
			Message: err.Error(),
		})
		return
	}

	response := GenerateEmbeddingResponse{
		Text:       req.Text,
		Embeddings: embeddings,
		Dimensions: len(embeddings),
	}

	c.JSON(http.StatusOK, response)
}

// VectorSearch performs semantic similarity search
// @Summary Vector similarity search
// @Description Perform semantic similarity search across documents and knowledge items
// @Tags embeddings
// @Accept json
// @Produce json
// @Param request body VectorSearchRequest true "Search parameters"
// @Success 200 {object} VectorSearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/search [post]
func (h *EmbeddingHandler) VectorSearch(c *gin.Context) {
	var req VectorSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Set defaults
	options := &embedding.SearchOptions{
		Limit:      req.Limit,
		Threshold:  req.Threshold,
		Collection: req.Collection,
		Filters:    req.Filters,
	}

	if options.Limit <= 0 {
		options.Limit = 10
	}
	if options.Threshold <= 0 {
		options.Threshold = 0.7
	}

	results, err := h.service.VectorSearch(c.Request.Context(), req.Query, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to perform vector search",
			Message: err.Error(),
		})
		return
	}

	response := VectorSearchResponse{
		Query:   req.Query,
		Results: results,
		Count:   len(results),
	}

	c.JSON(http.StatusOK, response)
}

// ProcessEmbeddings processes embeddings for documents and knowledge items
// @Summary Process embeddings
// @Description Generate embeddings for documents and knowledge items in batch
// @Tags embeddings
// @Accept json
// @Produce json
// @Param request body ProcessEmbeddingsRequest true "Processing parameters"
// @Success 200 {object} ProcessEmbeddingsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/process [post]
func (h *EmbeddingHandler) ProcessEmbeddings(c *gin.Context) {
	var req ProcessEmbeddingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	var result *embedding.ProcessResult
	var err error

	if req.ProcessAll {
		// Process all documents and knowledge items without embeddings
		docResult, docErr := h.pipeline.ProcessAllDocuments(c.Request.Context())
		knowledgeResult, knowledgeErr := h.pipeline.ProcessAllKnowledgeItems(c.Request.Context())

		// Combine results
		result = &embedding.ProcessResult{
			TotalProcessed: 0,
			Successful:     0,
			Failed:         0,
			Errors:         []string{},
		}

		if docErr == nil && docResult != nil {
			result.TotalProcessed += docResult.TotalProcessed
			result.Successful += docResult.Successful
			result.Failed += docResult.Failed
			result.Errors = append(result.Errors, docResult.Errors...)
		} else if docErr != nil {
			result.Errors = append(result.Errors, "Document processing failed: "+docErr.Error())
		}

		if knowledgeErr == nil && knowledgeResult != nil {
			result.TotalProcessed += knowledgeResult.TotalProcessed
			result.Successful += knowledgeResult.Successful
			result.Failed += knowledgeResult.Failed
			result.Errors = append(result.Errors, knowledgeResult.Errors...)
		} else if knowledgeErr != nil {
			result.Errors = append(result.Errors, "Knowledge processing failed: "+knowledgeErr.Error())
		}

		if len(result.Errors) > 0 && result.TotalProcessed == 0 {
			err = errors.New("processing failed")
		}
	} else {
		// Process specific items
		if len(req.DocumentIDs) > 0 {
			var documentIDs []primitive.ObjectID
			for _, idStr := range req.DocumentIDs {
				id, parseErr := primitive.ObjectIDFromHex(idStr)
				if parseErr != nil {
					c.JSON(http.StatusBadRequest, ErrorResponse{
						Error:   "Invalid document ID",
						Message: parseErr.Error(),
					})
					return
				}
				documentIDs = append(documentIDs, id)
			}

			result, err = h.pipeline.ProcessSpecificDocuments(c.Request.Context(), documentIDs)
		} else if len(req.KnowledgeIDs) > 0 {
			var knowledgeIDs []primitive.ObjectID
			for _, idStr := range req.KnowledgeIDs {
				id, parseErr := primitive.ObjectIDFromHex(idStr)
				if parseErr != nil {
					c.JSON(http.StatusBadRequest, ErrorResponse{
						Error:   "Invalid knowledge ID",
						Message: parseErr.Error(),
					})
					return
				}
				knowledgeIDs = append(knowledgeIDs, id)
			}

			result, err = h.pipeline.ProcessSpecificKnowledgeItems(c.Request.Context(), knowledgeIDs)
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid request",
				Message: "Must specify document_ids, knowledge_ids, or process_all",
			})
			return
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process embeddings",
			Message: err.Error(),
		})
		return
	}

	response := ProcessEmbeddingsResponse{
		TotalProcessed: result.TotalProcessed,
		Successful:     result.Successful,
		Failed:         result.Failed,
		Duration:       result.Duration.String(),
		Errors:         result.Errors,
	}

	c.JSON(http.StatusOK, response)
}

// GetEmbeddingStats returns embedding statistics
// @Summary Get embedding statistics
// @Description Get statistics about embeddings in the database
// @Tags embeddings
// @Produce json
// @Success 200 {object} embedding.EmbeddingStats
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/stats [get]
func (h *EmbeddingHandler) GetEmbeddingStats(c *gin.Context) {
	stats, err := h.repository.GetEmbeddingStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get embedding stats",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GenerateDocumentEmbedding generates embedding for a specific document
// @Summary Generate document embedding
// @Description Generate embedding for a specific document by ID
// @Tags embeddings
// @Param id path string true "Document ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/documents/{id}/embedding [post]
func (h *EmbeddingHandler) GenerateDocumentEmbedding(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid document ID",
			Message: err.Error(),
		})
		return
	}

	err = h.service.GenerateDocumentEmbedding(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate document embedding",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Document embedding generated successfully",
	})
}

// GenerateKnowledgeEmbedding generates embedding for a specific knowledge item
// @Summary Generate knowledge embedding
// @Description Generate embedding for a specific knowledge item by ID
// @Tags embeddings
// @Param id path string true "Knowledge Item ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/knowledge/{id}/embedding [post]
func (h *EmbeddingHandler) GenerateKnowledgeEmbedding(c *gin.Context) {
	idStr := c.Param("id")
	knowledgeID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid knowledge ID",
			Message: err.Error(),
		})
		return
	}

	err = h.service.GenerateKnowledgeEmbedding(c.Request.Context(), knowledgeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate knowledge embedding",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Knowledge embedding generated successfully",
	})
}

// GetSimilarDocuments finds documents similar to a given document
// @Summary Get similar documents
// @Description Find documents similar to a given document
// @Tags embeddings
// @Param id path string true "Document ID"
// @Param limit query int false "Maximum number of results" default(5)
// @Success 200 {object} VectorSearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/documents/{id}/similar [get]
func (h *EmbeddingHandler) GetSimilarDocuments(c *gin.Context) {
	idStr := c.Param("id")
	documentID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid document ID",
			Message: err.Error(),
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 5
	}

	results, err := h.service.GetSimilarDocuments(c.Request.Context(), documentID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to find similar documents",
			Message: err.Error(),
		})
		return
	}

	response := VectorSearchResponse{
		Query:   "Similar to document " + idStr,
		Results: results,
		Count:   len(results),
	}

	c.JSON(http.StatusOK, response)
}

// GetSimilarKnowledge finds knowledge items similar to a given knowledge item
// @Summary Get similar knowledge items
// @Description Find knowledge items similar to a given knowledge item
// @Tags embeddings
// @Param id path string true "Knowledge Item ID"
// @Param limit query int false "Maximum number of results" default(5)
// @Success 200 {object} VectorSearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/knowledge/{id}/similar [get]
func (h *EmbeddingHandler) GetSimilarKnowledge(c *gin.Context) {
	idStr := c.Param("id")
	knowledgeID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid knowledge ID",
			Message: err.Error(),
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 5
	}

	results, err := h.service.GetSimilarKnowledge(c.Request.Context(), knowledgeID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to find similar knowledge items",
			Message: err.Error(),
		})
		return
	}

	response := VectorSearchResponse{
		Query:   "Similar to knowledge item " + idStr,
		Results: results,
		Count:   len(results),
	}

	c.JSON(http.StatusOK, response)
}

// ClearCache clears the embedding cache
// @Summary Clear embedding cache
// @Description Clear all cached embeddings from Redis
// @Tags embeddings
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/embeddings/cache [delete]
func (h *EmbeddingHandler) ClearCache(c *gin.Context) {
	err := h.service.ClearCache(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to clear cache",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Embedding cache cleared successfully",
	})
}
