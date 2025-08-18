package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-government-consultant/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditHandler handles audit and reporting API endpoints
type AuditHandler struct {
	auditService AuditServiceInterface
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService AuditServiceInterface) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// AuditLogRequest represents an audit log search request
type AuditLogRequest struct {
	UserID     string    `form:"user_id"`
	Action     string    `form:"action"`
	Resource   string    `form:"resource"`
	Result     string    `form:"result"`
	IPAddress  string    `form:"ip_address"`
	DateFrom   string    `form:"date_from"`
	DateTo     string    `form:"date_to"`
	Limit      int       `form:"limit"`
	Skip       int       `form:"skip"`
	SortBy     string    `form:"sort_by"`
	SortOrder  string    `form:"sort_order"`
}

// AuditReportRequest represents an audit report generation request
type AuditReportRequest struct {
	ReportType  string    `json:"report_type" binding:"required"`
	DateFrom    string    `json:"date_from" binding:"required"`
	DateTo      string    `json:"date_to" binding:"required"`
	UserID      string    `json:"user_id,omitempty"`
	Department  string    `json:"department,omitempty"`
	Actions     []string  `json:"actions,omitempty"`
	Resources   []string  `json:"resources,omitempty"`
	Format      string    `json:"format,omitempty"`
	IncludeDetails bool   `json:"include_details,omitempty"`
}

// DataLineageRequest represents a data lineage tracking request
type DataLineageRequest struct {
	ResourceID   string `form:"resource_id" binding:"required"`
	ResourceType string `form:"resource_type" binding:"required"`
	Depth        int    `form:"depth"`
}

// GetAuditLogs retrieves audit logs based on search criteria
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
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

	// Check permissions (admin or audit role required)
	if !user.IsAdmin() && !user.HasPermission("audit", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view audit logs",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse search parameters
	var req AuditLogRequest
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
		req.Limit = 50
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}
	if req.Skip < 0 {
		req.Skip = 0
	}

	// Parse date filters
	var dateFrom, dateTo *time.Time
	if req.DateFrom != "" {
		if parsed, err := time.Parse(time.RFC3339, req.DateFrom); err == nil {
			dateFrom = &parsed
		}
	}
	if req.DateTo != "" {
		if parsed, err := time.Parse(time.RFC3339, req.DateTo); err == nil {
			dateTo = &parsed
		}
	}

	// Create audit query
	query := &AuditQuery{
		UserID:    req.UserID,
		Action:    req.Action,
		Resource:  req.Resource,
		Result:    req.Result,
		IPAddress: req.IPAddress,
		DateFrom:  dateFrom,
		DateTo:    dateTo,
		Limit:     req.Limit,
		Skip:      req.Skip,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Search audit logs
	logs, err := h.auditService.SearchAuditLogs(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve audit logs",
			Message: err.Error(),
			Code:    "AUDIT_SEARCH_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": logs,
		"total":      len(logs),
		"limit":      req.Limit,
		"skip":       req.Skip,
	})
}

// GetAuditLog retrieves a specific audit log entry by ID
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	logID := c.Param("id")
	if logID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Audit log ID is required",
			Code:  "MISSING_LOG_ID",
		})
		return
	}

	// Validate ObjectID
	objID, err := primitive.ObjectIDFromHex(logID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid audit log ID format",
			Message: err.Error(),
			Code:    "INVALID_LOG_ID",
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
	if !user.IsAdmin() && !user.HasPermission("audit", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view audit logs",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Get audit log
	auditLog, err := h.auditService.GetAuditLog(c.Request.Context(), objID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Audit log not found",
				Message: err.Error(),
				Code:    "LOG_NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve audit log",
			Message: err.Error(),
			Code:    "LOG_RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_log": auditLog,
	})
}

// GenerateAuditReport generates an audit report based on criteria
func (h *AuditHandler) GenerateAuditReport(c *gin.Context) {
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

	// Check permissions (admin or audit manager required)
	if !user.IsAdmin() && !user.HasPermission("audit", "report") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to generate audit reports",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req AuditReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Parse dates
	dateFrom, err := time.Parse(time.RFC3339, req.DateFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid date_from format",
			Message: "Use RFC3339 format (e.g., 2023-01-01T00:00:00Z)",
			Code:    "INVALID_DATE_FORMAT",
		})
		return
	}

	dateTo, err := time.Parse(time.RFC3339, req.DateTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid date_to format",
			Message: "Use RFC3339 format (e.g., 2023-12-31T23:59:59Z)",
			Code:    "INVALID_DATE_FORMAT",
		})
		return
	}

	// Validate date range
	if dateTo.Before(dateFrom) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "date_to must be after date_from",
			Code:  "INVALID_DATE_RANGE",
		})
		return
	}

	// Set default format
	if req.Format == "" {
		req.Format = "json"
	}

	// Create audit criteria
	criteria := &AuditCriteria{
		ReportType:     req.ReportType,
		DateFrom:       dateFrom,
		DateTo:         dateTo,
		UserID:         req.UserID,
		Department:     req.Department,
		Actions:        req.Actions,
		Resources:      req.Resources,
		Format:         req.Format,
		IncludeDetails: req.IncludeDetails,
		GeneratedBy:    user.ID.Hex(),
	}

	// Generate audit report
	report, err := h.auditService.GenerateAuditReport(c.Request.Context(), criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate audit report",
			Message: err.Error(),
			Code:    "REPORT_GENERATION_FAILED",
		})
		return
	}

	// Set appropriate content type based on format
	switch req.Format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=audit_report.csv")
	case "pdf":
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename=audit_report.pdf")
	default:
		c.Header("Content-Type", "application/json")
	}

	c.JSON(http.StatusOK, gin.H{
		"report": report,
	})
}

// TrackDataLineage tracks the lineage of a specific data resource
func (h *AuditHandler) TrackDataLineage(c *gin.Context) {
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
	if !user.IsAdmin() && !user.HasPermission("audit", "read") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to track data lineage",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse request parameters
	var req DataLineageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Set default depth
	if req.Depth <= 0 {
		req.Depth = 3
	}
	if req.Depth > 10 {
		req.Depth = 10
	}

	// Track data lineage
	lineage, err := h.auditService.TrackDataLineage(c.Request.Context(), req.ResourceID, req.ResourceType, req.Depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to track data lineage",
			Message: err.Error(),
			Code:    "LINEAGE_TRACKING_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"lineage": lineage,
	})
}

// GetUserActivity retrieves activity logs for a specific user
func (h *AuditHandler) GetUserActivity(c *gin.Context) {
	targetUserID := c.Param("user_id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "User ID is required",
			Code:  "MISSING_USER_ID",
		})
		return
	}

	// Validate ObjectID
	targetObjID, err := primitive.ObjectIDFromHex(targetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid user ID format",
			Message: err.Error(),
			Code:    "INVALID_USER_ID",
		})
		return
	}

	// Get current user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "User not authenticated",
			Code:  "NOT_AUTHENTICATED",
		})
		return
	}

	user := userInterface.(*models.User)

	// Check permissions (admin, audit role, or own activity)
	if !user.IsAdmin() && !user.HasPermission("audit", "read") && user.ID != targetObjID {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view user activity",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse pagination and filter parameters
	limitStr := c.DefaultQuery("limit", "50")
	skipStr := c.DefaultQuery("skip", "0")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	action := c.Query("action")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	skip, err := strconv.Atoi(skipStr)
	if err != nil || skip < 0 {
		skip = 0
	}

	// Create query for user activity
	query := &AuditQuery{
		UserID: targetUserID,
		Action: action,
		Limit:  limit,
		Skip:   skip,
	}

	// Parse date filters
	if dateFrom != "" {
		if parsed, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			query.DateFrom = &parsed
		}
	}
	if dateTo != "" {
		if parsed, err := time.Parse(time.RFC3339, dateTo); err == nil {
			query.DateTo = &parsed
		}
	}

	// Get user activity
	activity, err := h.auditService.GetUserActivity(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve user activity",
			Message: err.Error(),
			Code:    "ACTIVITY_RETRIEVAL_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  targetUserID,
		"activity": activity,
		"total":    len(activity),
		"limit":    limit,
		"skip":     skip,
	})
}

// GetSystemActivity retrieves system-wide activity statistics
func (h *AuditHandler) GetSystemActivity(c *gin.Context) {
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

	// Check permissions (admin only)
	if !user.IsAdmin() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to view system activity",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse time range parameters
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	granularity := c.DefaultQuery("granularity", "day") // hour, day, week, month

	// Validate granularity
	validGranularities := map[string]bool{
		"hour":  true,
		"day":   true,
		"week":  true,
		"month": true,
	}
	if !validGranularities[granularity] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid granularity",
			Message: "Valid values: hour, day, week, month",
			Code:    "INVALID_GRANULARITY",
		})
		return
	}

	// Set default date range if not provided (last 30 days)
	var dateFromParsed, dateToParsed time.Time
	if dateFrom == "" {
		dateFromParsed = time.Now().AddDate(0, 0, -30)
	} else {
		var err error
		dateFromParsed, err = time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid date_from format",
				Message: "Use RFC3339 format",
				Code:    "INVALID_DATE_FORMAT",
			})
			return
		}
	}

	if dateTo == "" {
		dateToParsed = time.Now()
	} else {
		var err error
		dateToParsed, err = time.Parse(time.RFC3339, dateTo)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid date_to format",
				Message: "Use RFC3339 format",
				Code:    "INVALID_DATE_FORMAT",
			})
			return
		}
	}

	// Get system activity statistics
	stats, err := h.auditService.GetSystemActivity(c.Request.Context(), dateFromParsed, dateToParsed, granularity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve system activity",
			Message: err.Error(),
			Code:    "SYSTEM_ACTIVITY_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics":  stats,
		"date_from":   dateFromParsed,
		"date_to":     dateToParsed,
		"granularity": granularity,
	})
}

// GetComplianceReport generates a compliance report
func (h *AuditHandler) GetComplianceReport(c *gin.Context) {
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

	// Check permissions (admin or compliance officer)
	if !user.IsAdmin() && !user.HasPermission("audit", "compliance") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to generate compliance reports",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse parameters
	standard := c.Query("standard") // e.g., "SOX", "GDPR", "HIPAA", "FedRAMP"
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	format := c.DefaultQuery("format", "json")

	if standard == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Compliance standard is required",
			Code:  "MISSING_STANDARD",
		})
		return
	}

	// Set default date range (last 90 days)
	var dateFromParsed, dateToParsed time.Time
	if dateFrom == "" {
		dateFromParsed = time.Now().AddDate(0, 0, -90)
	} else {
		var err error
		dateFromParsed, err = time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid date_from format",
				Message: "Use RFC3339 format",
				Code:    "INVALID_DATE_FORMAT",
			})
			return
		}
	}

	if dateTo == "" {
		dateToParsed = time.Now()
	} else {
		var err error
		dateToParsed, err = time.Parse(time.RFC3339, dateTo)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid date_to format",
				Message: "Use RFC3339 format",
				Code:    "INVALID_DATE_FORMAT",
			})
			return
		}
	}

	// Generate compliance report
	report, err := h.auditService.GenerateComplianceReport(c.Request.Context(), standard, dateFromParsed, dateToParsed, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate compliance report",
			Message: err.Error(),
			Code:    "COMPLIANCE_REPORT_FAILED",
		})
		return
	}

	// Set appropriate content type
	switch format {
	case "pdf":
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename=compliance_report.pdf")
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=compliance_report.csv")
	default:
		c.Header("Content-Type", "application/json")
	}

	c.JSON(http.StatusOK, gin.H{
		"report": report,
	})
}

// ExportAuditLogs exports audit logs in various formats
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
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

	// Check permissions (admin or audit export role)
	if !user.IsAdmin() && !user.HasPermission("audit", "export") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Insufficient permissions to export audit logs",
			Code:  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Parse export parameters
	format := c.DefaultQuery("format", "csv")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	userID := c.Query("user_id")
	action := c.Query("action")
	resource := c.Query("resource")

	// Validate format
	validFormats := map[string]bool{
		"csv":  true,
		"json": true,
		"xml":  true,
	}
	if !validFormats[format] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid export format",
			Message: "Valid formats: csv, json, xml",
			Code:    "INVALID_FORMAT",
		})
		return
	}

	// Create export criteria
	criteria := &ExportCriteria{
		Format:   format,
		UserID:   userID,
		Action:   action,
		Resource: resource,
	}

	// Parse date filters
	if dateFrom != "" {
		if parsed, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			criteria.DateFrom = &parsed
		}
	}
	if dateTo != "" {
		if parsed, err := time.Parse(time.RFC3339, dateTo); err == nil {
			criteria.DateTo = &parsed
		}
	}

	// Export audit logs
	exportData, err := h.auditService.ExportAuditLogs(c.Request.Context(), criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to export audit logs",
			Message: err.Error(),
			Code:    "EXPORT_FAILED",
		})
		return
	}

	// Set appropriate headers
	filename := "audit_logs." + format
	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
	case "xml":
		c.Header("Content-Type", "application/xml")
	default:
		c.Header("Content-Type", "application/json")
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)

	c.Data(http.StatusOK, c.GetHeader("Content-Type"), exportData)
}