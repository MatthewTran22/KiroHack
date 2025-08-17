package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ai-government-consultant/internal/audit"
)

// AuditHandler handles HTTP requests for audit operations
type AuditHandler struct {
	auditService audit.Service
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService audit.Service) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// SearchAuditLogs handles GET /api/audit/logs
func (h *AuditHandler) SearchAuditLogs(c *gin.Context) {
	var query audit.AuditQuery

	// Parse query parameters
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			query.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			query.EndDate = &endDate
		}
	}

	if eventTypesStr := c.QueryArray("event_types"); len(eventTypesStr) > 0 {
		eventTypes := make([]audit.AuditEventType, len(eventTypesStr))
		for i, et := range eventTypesStr {
			eventTypes[i] = audit.AuditEventType(et)
		}
		query.EventTypes = eventTypes
	}

	if levelsStr := c.QueryArray("levels"); len(levelsStr) > 0 {
		levels := make([]audit.AuditLevel, len(levelsStr))
		for i, l := range levelsStr {
			levels[i] = audit.AuditLevel(l)
		}
		query.Levels = levels
	}

	if userIDs := c.QueryArray("user_ids"); len(userIDs) > 0 {
		query.UserIDs = userIDs
	}

	if resources := c.QueryArray("resources"); len(resources) > 0 {
		query.Resources = resources
	}

	if results := c.QueryArray("results"); len(results) > 0 {
		query.Results = results
	}

	if ipAddresses := c.QueryArray("ip_addresses"); len(ipAddresses) > 0 {
		query.IPAddresses = ipAddresses
	}

	if searchText := c.Query("search_text"); searchText != "" {
		query.SearchText = &searchText
	}

	// Parse pagination parameters
	query.Limit = 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}

	query.Offset = 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	// Parse sort parameters
	query.SortBy = c.DefaultQuery("sort_by", "timestamp")
	query.SortOrder = c.DefaultQuery("sort_order", "desc")

	// Execute search
	entries, total, err := h.auditService.SearchAuditLogs(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search audit logs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"limit":   query.Limit,
		"offset":  query.Offset,
	})
}

// GetAuditEntry handles GET /api/audit/logs/:id
func (h *AuditHandler) GetAuditEntry(c *gin.Context) {
	entryID := c.Param("id")
	if entryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entry ID is required"})
		return
	}

	entry, err := h.auditService.GetAuditEntry(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Audit entry not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// GetUserActivity handles GET /api/audit/users/:user_id/activity
func (h *AuditHandler) GetUserActivity(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Parse date range
	startDate := time.Now().AddDate(0, 0, -30) // Default: last 30 days
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = parsed
		}
	}

	endDate := time.Now()
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = parsed
		}
	}

	entries, err := h.auditService.GetUserActivity(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user activity",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    userID,
		"start_date": startDate,
		"end_date":   endDate,
		"entries":    entries,
		"count":      len(entries),
	})
}

// GetDataLineage handles GET /api/audit/lineage/:data_id
func (h *AuditHandler) GetDataLineage(c *gin.Context) {
	dataID := c.Param("data_id")
	if dataID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data ID is required"})
		return
	}

	lineage, err := h.auditService.GetDataLineage(c.Request.Context(), dataID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get data lineage",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data_id": dataID,
		"lineage": lineage,
	})
}

// GetDataProvenance handles GET /api/audit/provenance/:data_id
func (h *AuditHandler) GetDataProvenance(c *gin.Context) {
	dataID := c.Param("data_id")
	if dataID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data ID is required"})
		return
	}

	provenance, err := h.auditService.GetDataProvenance(c.Request.Context(), dataID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Data provenance not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data_id":    dataID,
		"provenance": provenance,
	})
}

// GenerateAuditReport handles POST /api/audit/reports
func (h *AuditHandler) GenerateAuditReport(c *gin.Context) {
	var request struct {
		StartDate  time.Time              `json:"start_date" binding:"required"`
		EndDate    time.Time              `json:"end_date" binding:"required"`
		EventTypes []audit.AuditEventType `json:"event_types,omitempty"`
		Levels     []audit.AuditLevel     `json:"levels,omitempty"`
		UserIDs    []string               `json:"user_ids,omitempty"`
		Resources  []string               `json:"resources,omitempty"`
		Format     string                 `json:"format" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate format
	if request.Format != "json" && request.Format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format must be 'json' or 'csv'"})
		return
	}

	// Build query
	query := audit.AuditQuery{
		StartDate:  &request.StartDate,
		EndDate:    &request.EndDate,
		EventTypes: request.EventTypes,
		Levels:     request.Levels,
		UserIDs:    request.UserIDs,
		Resources:  request.Resources,
		Limit:      10000, // Large limit for reports
		SortBy:     "timestamp",
		SortOrder:  "desc",
	}

	report, err := h.auditService.GenerateAuditReport(c.Request.Context(), query, request.Format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate audit report",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// GetAuditReport handles GET /api/audit/reports/:id
func (h *AuditHandler) GetAuditReport(c *gin.Context) {
	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
		return
	}

	report, err := h.auditService.GetAuditReport(c.Request.Context(), reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Audit report not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ListAuditReports handles GET /api/audit/reports
func (h *AuditHandler) ListAuditReports(c *gin.Context) {
	limit := 20 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	reports, err := h.auditService.ListAuditReports(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list audit reports",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"limit":   limit,
		"offset":  offset,
	})
}

// ExportAuditReport handles GET /api/audit/reports/:id/export
func (h *AuditHandler) ExportAuditReport(c *gin.Context) {
	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
		return
	}

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format must be 'json' or 'csv'"})
		return
	}

	data, err := h.auditService.ExportAuditReport(c.Request.Context(), reportID, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to export audit report",
			"details": err.Error(),
		})
		return
	}

	// Set appropriate content type and headers
	var contentType string
	var filename string

	switch format {
	case "json":
		contentType = "application/json"
		filename = "audit_report_" + reportID + ".json"
	case "csv":
		contentType = "text/csv"
		filename = "audit_report_" + reportID + ".csv"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}

// GetSecurityAlerts handles GET /api/audit/security/alerts
func (h *AuditHandler) GetSecurityAlerts(c *gin.Context) {
	status := c.Query("status") // Optional filter by status

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	alerts, err := h.auditService.GetSecurityAlerts(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get security alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateSecurityAlert handles PUT /api/audit/security/alerts/:id
func (h *AuditHandler) UpdateSecurityAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alert ID is required"})
		return
	}

	var request struct {
		Status     string  `json:"status" binding:"required"`
		Resolution *string `json:"resolution,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate status
	validStatuses := []string{"open", "investigating", "resolved", "false_positive"}
	isValidStatus := false
	for _, status := range validStatuses {
		if request.Status == status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Invalid status",
			"valid_statuses": validStatuses,
		})
		return
	}

	err := h.auditService.UpdateSecurityAlert(c.Request.Context(), alertID, request.Status, request.Resolution)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update security alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Security alert updated successfully",
		"alert_id": alertID,
		"status":   request.Status,
	})
}

// DetectAnomalies handles POST /api/audit/security/anomalies
func (h *AuditHandler) DetectAnomalies(c *gin.Context) {
	var request struct {
		UserID     string `json:"user_id" binding:"required"`
		TimeWindow string `json:"time_window"` // e.g., "5m", "1h", "24h"
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Parse time window
	timeWindow := 5 * time.Minute // Default
	if request.TimeWindow != "" {
		if tw, err := time.ParseDuration(request.TimeWindow); err == nil {
			timeWindow = tw
		}
	}

	alerts, err := h.auditService.DetectAnomalies(c.Request.Context(), request.UserID, timeWindow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to detect anomalies",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     request.UserID,
		"time_window": timeWindow.String(),
		"alerts":      alerts,
		"count":       len(alerts),
	})
}

// GenerateComplianceReport handles POST /api/audit/compliance/reports
func (h *AuditHandler) GenerateComplianceReport(c *gin.Context) {
	var request struct {
		Standard  string    `json:"standard" binding:"required"`
		StartDate time.Time `json:"start_date" binding:"required"`
		EndDate   time.Time `json:"end_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	period := audit.ReportPeriod{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
	}

	report, err := h.auditService.GenerateComplianceReport(c.Request.Context(), request.Standard, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate compliance report",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// GetComplianceReport handles GET /api/audit/compliance/reports/:id
func (h *AuditHandler) GetComplianceReport(c *gin.Context) {
	reportID := c.Param("id")
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
		return
	}

	report, err := h.auditService.GetComplianceReport(c.Request.Context(), reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Compliance report not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}

// ValidateCompliance handles GET /api/audit/compliance/validate/:standard
func (h *AuditHandler) ValidateCompliance(c *gin.Context) {
	standard := c.Param("standard")
	if standard == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Compliance standard is required"})
		return
	}

	report, err := h.auditService.ValidateCompliance(c.Request.Context(), standard)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to validate compliance",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}
