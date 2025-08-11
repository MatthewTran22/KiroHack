package models

import "errors"

// Document validation errors
var (
	ErrDocumentNameRequired        = errors.New("document name is required")
	ErrDocumentContentTypeRequired = errors.New("document content type is required")
	ErrDocumentSizeInvalid         = errors.New("document size must be greater than 0")
	ErrDocumentUploadedByRequired  = errors.New("document uploaded_by is required")
)

// User validation errors
var (
	ErrUserEmailRequired             = errors.New("user email is required")
	ErrUserNameRequired              = errors.New("user name is required")
	ErrUserDepartmentRequired        = errors.New("user department is required")
	ErrUserRoleRequired              = errors.New("user role is required")
	ErrUserSecurityClearanceRequired = errors.New("user security clearance is required")
)

// Consultation validation errors
var (
	ErrConsultationUserIDRequired = errors.New("consultation user_id is required")
	ErrConsultationQueryRequired  = errors.New("consultation query is required")
	ErrConsultationTypeRequired   = errors.New("consultation type is required")
)

// Knowledge validation errors
var (
	ErrKnowledgeContentRequired   = errors.New("knowledge content is required")
	ErrKnowledgeTypeRequired      = errors.New("knowledge type is required")
	ErrKnowledgeTitleRequired     = errors.New("knowledge title is required")
	ErrKnowledgeCreatedByRequired = errors.New("knowledge created_by is required")
	ErrKnowledgeConfidenceInvalid = errors.New("knowledge confidence must be between 0.0 and 1.0")
)

// Database errors
var (
	ErrDatabaseConnection   = errors.New("database connection failed")
	ErrDatabaseTimeout      = errors.New("database operation timed out")
	ErrDocumentNotFound     = errors.New("document not found")
	ErrUserNotFound         = errors.New("user not found")
	ErrConsultationNotFound = errors.New("consultation not found")
	ErrKnowledgeNotFound    = errors.New("knowledge item not found")
	ErrDuplicateEntry       = errors.New("duplicate entry")
)
