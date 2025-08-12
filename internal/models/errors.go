package models

import "errors"

// User validation errors
var (
	ErrUserEmailRequired             = errors.New("user email is required")
	ErrUserNameRequired              = errors.New("user name is required")
	ErrUserDepartmentRequired        = errors.New("user department is required")
	ErrUserRoleRequired              = errors.New("user role is required")
	ErrUserSecurityClearanceRequired = errors.New("user security clearance is required")
	ErrUserNotFound                  = errors.New("user not found")
	ErrUserAlreadyExists             = errors.New("user already exists")
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrUserInactive                  = errors.New("user account is inactive")
)

// Authentication errors
var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrTokenBlacklisted    = errors.New("token is blacklisted")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrMFARequired         = errors.New("multi-factor authentication required")
	ErrInvalidMFACode      = errors.New("invalid MFA code")
)

// Authorization errors
var (
	ErrInsufficientPermissions  = errors.New("insufficient permissions")
	ErrAccessDenied             = errors.New("access denied")
	ErrInvalidSecurityClearance = errors.New("invalid security clearance for resource")
)

// Document validation errors
var (
	ErrDocumentNameRequired        = errors.New("document name is required")
	ErrDocumentContentTypeRequired = errors.New("document content type is required")
	ErrDocumentSizeInvalid         = errors.New("document size is invalid")
	ErrDocumentUploadedByRequired  = errors.New("document uploaded by is required")
)

// Consultation validation errors
var (
	ErrConsultationUserIDRequired = errors.New("consultation user ID is required")
	ErrConsultationQueryRequired  = errors.New("consultation query is required")
	ErrConsultationTypeRequired   = errors.New("consultation type is required")
)

// Knowledge validation errors
var (
	ErrKnowledgeContentRequired   = errors.New("knowledge content is required")
	ErrKnowledgeTypeRequired      = errors.New("knowledge type is required")
	ErrKnowledgeTitleRequired     = errors.New("knowledge title is required")
	ErrKnowledgeCreatedByRequired = errors.New("knowledge created by is required")
	ErrKnowledgeConfidenceInvalid = errors.New("knowledge confidence is invalid")
)

// Embedding validation errors
var (
	ErrEmbeddingRequired      = errors.New("embedding is required")
	ErrEmbeddingInvalid       = errors.New("embedding is invalid")
	ErrEmbeddingDimensionZero = errors.New("embedding dimension cannot be zero")
	ErrEmbeddingNotGenerated  = errors.New("embedding has not been generated")
)
