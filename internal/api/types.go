package api

import (
	"ai-government-consultant/internal/models"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Documents []*models.Document `json:"documents"`
	Total     int64              `json:"total"`
	Limit     int                `json:"limit"`
	Skip      int                `json:"skip"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
