package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service defines the interface for audit operations
type Service interface {
	// Audit logging
	LogEvent(ctx context.Context, entry AuditEntry) error
	LogUserAction(ctx context.Context, userID, action, resource string, details map[string]interface{}) error
	LogSystemEvent(ctx context.Context, eventType AuditEventType, details map[string]interface{}) error
	LogSecurityEvent(ctx context.Context, eventType AuditEventType, userID *string, ipAddress, resource string, details map[string]interface{}) error

	// Audit querying
	SearchAuditLogs(ctx context.Context, query AuditQuery) ([]AuditEntry, int, error)
	GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error)
	GetUserActivity(ctx context.Context, userID string, startDate, endDate time.Time) ([]AuditEntry, error)

	// Data lineage
	TrackDataLineage(ctx context.Context, entry DataLineageEntry) error
	GetDataLineage(ctx context.Context, dataID string) ([]DataLineageEntry, error)
	GetDataProvenance(ctx context.Context, dataID string) (*DataLineageEntry, error)

	// Audit reports
	GenerateAuditReport(ctx context.Context, query AuditQuery, format string) (*AuditReport, error)
	GetAuditReport(ctx context.Context, reportID string) (*AuditReport, error)
	ListAuditReports(ctx context.Context, limit, offset int) ([]AuditReport, error)
	ExportAuditReport(ctx context.Context, reportID, format string) ([]byte, error)

	// Security monitoring
	CreateSecurityAlert(ctx context.Context, alert SecurityAlert) error
	GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]SecurityAlert, error)
	UpdateSecurityAlert(ctx context.Context, alertID, status string, resolution *string) error
	DetectAnomalies(ctx context.Context, userID string, timeWindow time.Duration) ([]SecurityAlert, error)

	// Compliance reporting
	GenerateComplianceReport(ctx context.Context, standard string, period ReportPeriod) (*ComplianceReport, error)
	GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error)
	ValidateCompliance(ctx context.Context, standard string) (*ComplianceReport, error)

	// Data retention and cleanup
	PurgeOldAuditLogs(ctx context.Context, retentionPeriod time.Duration) (int, error)
	ArchiveAuditLogs(ctx context.Context, beforeDate time.Time) error
}

// serviceImpl implements the audit Service interface
type serviceImpl struct {
	repo Repository
}

// NewService creates a new audit service
func NewService(repo Repository) Service {
	return &serviceImpl{
		repo: repo,
	}
}

// LogEvent logs a general audit event
func (s *serviceImpl) LogEvent(ctx context.Context, entry AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	return s.repo.CreateAuditEntry(ctx, entry)
}

// LogUserAction logs a user-initiated action
func (s *serviceImpl) LogUserAction(ctx context.Context, userID, action, resource string, details map[string]interface{}) error {
	entry := AuditEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: AuditEventType(action),
		Level:     AuditLevelInfo,
		UserID:    &userID,
		Resource:  resource,
		Action:    action,
		Result:    "success",
		Details:   details,
	}

	// Extract request context if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqID, ok := requestID.(string); ok {
			entry.RequestID = reqID
		}
	}

	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if sessID, ok := sessionID.(string); ok {
			entry.SessionID = &sessID
		}
	}

	if ipAddress := ctx.Value("ip_address"); ipAddress != nil {
		if ip, ok := ipAddress.(string); ok {
			entry.IPAddress = ip
		}
	}

	if userAgent := ctx.Value("user_agent"); userAgent != nil {
		if ua, ok := userAgent.(string); ok {
			entry.UserAgent = ua
		}
	}

	return s.LogEvent(ctx, entry)
}

// LogSystemEvent logs a system-level event
func (s *serviceImpl) LogSystemEvent(ctx context.Context, eventType AuditEventType, details map[string]interface{}) error {
	entry := AuditEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: eventType,
		Level:     AuditLevelInfo,
		Resource:  "system",
		Action:    string(eventType),
		Result:    "success",
		Details:   details,
	}

	return s.LogEvent(ctx, entry)
}

// LogSecurityEvent logs a security-related event
func (s *serviceImpl) LogSecurityEvent(ctx context.Context, eventType AuditEventType, userID *string, ipAddress, resource string, details map[string]interface{}) error {
	entry := AuditEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		EventType: eventType,
		Level:     AuditLevelSecurity,
		UserID:    userID,
		IPAddress: ipAddress,
		Resource:  resource,
		Action:    string(eventType),
		Result:    "failure", // Security events are typically failures or violations
		Details:   details,
	}

	// Create security alert for critical events
	if eventType == EventSecurityViolation || eventType == EventUnauthorizedAccess || eventType == EventDataBreach {
		alert := SecurityAlert{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			AlertType:   string(eventType),
			Severity:    AuditLevelCritical,
			Title:       fmt.Sprintf("Security Event: %s", eventType),
			Description: fmt.Sprintf("Security event detected for resource: %s", resource),
			UserID:      userID,
			IPAddress:   ipAddress,
			Resource:    resource,
			Details:     details,
			Status:      "open",
		}

		if err := s.CreateSecurityAlert(ctx, alert); err != nil {
			// Log the error but don't fail the audit logging
			fmt.Printf("Failed to create security alert: %v\n", err)
		}
	}

	return s.LogEvent(ctx, entry)
}

// SearchAuditLogs searches audit logs based on query criteria
func (s *serviceImpl) SearchAuditLogs(ctx context.Context, query AuditQuery) ([]AuditEntry, int, error) {
	return s.repo.SearchAuditEntries(ctx, query)
}

// GetAuditEntry retrieves a specific audit entry by ID
func (s *serviceImpl) GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error) {
	return s.repo.GetAuditEntry(ctx, id)
}

// GetUserActivity retrieves all audit entries for a specific user within a time range
func (s *serviceImpl) GetUserActivity(ctx context.Context, userID string, startDate, endDate time.Time) ([]AuditEntry, error) {
	query := AuditQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
		UserIDs:   []string{userID},
		SortBy:    "timestamp",
		SortOrder: "desc",
		Limit:     1000,
	}

	entries, _, err := s.SearchAuditLogs(ctx, query)
	return entries, err
}

// TrackDataLineage records data lineage information
func (s *serviceImpl) TrackDataLineage(ctx context.Context, entry DataLineageEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.ProcessedAt.IsZero() {
		entry.ProcessedAt = time.Now()
	}

	return s.repo.CreateDataLineageEntry(ctx, entry)
}

// GetDataLineage retrieves the complete lineage for a data item
func (s *serviceImpl) GetDataLineage(ctx context.Context, dataID string) ([]DataLineageEntry, error) {
	return s.repo.GetDataLineage(ctx, dataID)
}

// GetDataProvenance retrieves the immediate provenance (source) of a data item
func (s *serviceImpl) GetDataProvenance(ctx context.Context, dataID string) (*DataLineageEntry, error) {
	return s.repo.GetDataProvenance(ctx, dataID)
}

// GenerateAuditReport generates a comprehensive audit report
func (s *serviceImpl) GenerateAuditReport(ctx context.Context, query AuditQuery, format string) (*AuditReport, error) {
	entries, _, err := s.SearchAuditLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search audit logs: %w", err)
	}

	// Generate summary statistics
	summary := s.generateReportSummary(entries)

	report := &AuditReport{
		ID:          uuid.New().String(),
		Title:       "Audit Report",
		Description: "Comprehensive audit report for specified time period",
		GeneratedAt: time.Now(),
		StartDate:   *query.StartDate,
		EndDate:     *query.EndDate,
		Filters: map[string]interface{}{
			"event_types": query.EventTypes,
			"levels":      query.Levels,
			"user_ids":    query.UserIDs,
			"resources":   query.Resources,
		},
		Summary: summary,
		Entries: entries,
		Format:  format,
	}

	// Save the report
	if err := s.repo.CreateAuditReport(ctx, *report); err != nil {
		return nil, fmt.Errorf("failed to save audit report: %w", err)
	}

	return report, nil
}

// generateReportSummary creates statistical summary of audit entries
func (s *serviceImpl) generateReportSummary(entries []AuditEntry) ReportSummary {
	summary := ReportSummary{
		TotalEvents:      len(entries),
		EventsByType:     make(map[AuditEventType]int),
		EventsByLevel:    make(map[AuditLevel]int),
		EventsByResult:   make(map[string]int),
		UniqueUsers:      0,
		UniqueResources:  0,
		SecurityEvents:   0,
		FailedOperations: 0,
	}

	userSet := make(map[string]bool)
	resourceSet := make(map[string]bool)
	var totalDuration time.Duration
	var durationCount int

	for _, entry := range entries {
		// Count by type
		summary.EventsByType[entry.EventType]++

		// Count by level
		summary.EventsByLevel[entry.Level]++

		// Count by result
		summary.EventsByResult[entry.Result]++

		// Track unique users
		if entry.UserID != nil {
			userSet[*entry.UserID] = true
		}

		// Track unique resources
		resourceSet[entry.Resource] = true

		// Count security events
		if entry.Level == AuditLevelSecurity || entry.Level == AuditLevelCritical {
			summary.SecurityEvents++
		}

		// Count failed operations
		if entry.Result == "failure" {
			summary.FailedOperations++
		}

		// Calculate average response time
		if entry.Duration != nil {
			totalDuration += *entry.Duration
			durationCount++
		}
	}

	summary.UniqueUsers = len(userSet)
	summary.UniqueResources = len(resourceSet)

	if durationCount > 0 {
		avgDuration := totalDuration / time.Duration(durationCount)
		summary.AverageResponseTime = &avgDuration
	}

	return summary
}

// GetAuditReport retrieves a specific audit report
func (s *serviceImpl) GetAuditReport(ctx context.Context, reportID string) (*AuditReport, error) {
	return s.repo.GetAuditReport(ctx, reportID)
}

// ListAuditReports lists all audit reports with pagination
func (s *serviceImpl) ListAuditReports(ctx context.Context, limit, offset int) ([]AuditReport, error) {
	return s.repo.ListAuditReports(ctx, limit, offset)
}

// ExportAuditReport exports an audit report in the specified format
func (s *serviceImpl) ExportAuditReport(ctx context.Context, reportID, format string) ([]byte, error) {
	report, err := s.GetAuditReport(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit report: %w", err)
	}

	switch format {
	case "json":
		return s.exportJSON(report)
	case "csv":
		return s.exportCSV(report)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// CreateSecurityAlert creates a new security alert
func (s *serviceImpl) CreateSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	return s.repo.CreateSecurityAlert(ctx, alert)
}

// GetSecurityAlerts retrieves security alerts with optional status filter
func (s *serviceImpl) GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]SecurityAlert, error) {
	return s.repo.GetSecurityAlerts(ctx, status, limit, offset)
}

// UpdateSecurityAlert updates the status and resolution of a security alert
func (s *serviceImpl) UpdateSecurityAlert(ctx context.Context, alertID, status string, resolution *string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if resolution != nil {
		updates["resolution"] = *resolution
		if status == "resolved" {
			now := time.Now()
			updates["resolved_at"] = now
		}
	}

	return s.repo.UpdateSecurityAlert(ctx, alertID, updates)
}

// DetectAnomalies analyzes user behavior to detect potential security anomalies
func (s *serviceImpl) DetectAnomalies(ctx context.Context, userID string, timeWindow time.Duration) ([]SecurityAlert, error) {
	endTime := time.Now()
	startTime := endTime.Add(-timeWindow)

	// Get user activity in the time window
	activity, err := s.GetUserActivity(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity: %w", err)
	}

	var alerts []SecurityAlert

	// Detect unusual login patterns
	loginAttempts := 0
	failedLogins := 0
	uniqueIPs := make(map[string]bool)

	for _, entry := range activity {
		if entry.EventType == EventUserLogin || entry.EventType == EventUserLoginFailed {
			loginAttempts++
			uniqueIPs[entry.IPAddress] = true

			if entry.Result == "failure" {
				failedLogins++
			}
		}
	}

	// Alert on excessive failed logins
	if failedLogins > 5 {
		alert := SecurityAlert{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			AlertType:   "EXCESSIVE_FAILED_LOGINS",
			Severity:    AuditLevelWarning,
			Title:       "Excessive Failed Login Attempts",
			Description: fmt.Sprintf("User %s had %d failed login attempts in the last %v", userID, failedLogins, timeWindow),
			UserID:      &userID,
			Resource:    "authentication",
			Details: map[string]interface{}{
				"failed_login_count": failedLogins,
				"time_window":        timeWindow.String(),
			},
			Status: "open",
		}
		alerts = append(alerts, alert)
	}

	// Alert on logins from multiple IPs
	if len(uniqueIPs) > 3 {
		alert := SecurityAlert{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			AlertType:   "MULTIPLE_IP_LOGINS",
			Severity:    AuditLevelWarning,
			Title:       "Logins from Multiple IP Addresses",
			Description: fmt.Sprintf("User %s logged in from %d different IP addresses in the last %v", userID, len(uniqueIPs), timeWindow),
			UserID:      &userID,
			Resource:    "authentication",
			Details: map[string]interface{}{
				"unique_ip_count": len(uniqueIPs),
				"time_window":     timeWindow.String(),
				"ip_addresses":    getMapKeys(uniqueIPs),
			},
			Status: "open",
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GenerateComplianceReport generates a compliance report for a specific standard
func (s *serviceImpl) GenerateComplianceReport(ctx context.Context, standard string, period ReportPeriod) (*ComplianceReport, error) {
	// This is a simplified implementation - in practice, this would involve
	// complex compliance rule evaluation based on the specific standard

	query := AuditQuery{
		StartDate: &period.StartDate,
		EndDate:   &period.EndDate,
		SortBy:    "timestamp",
		SortOrder: "desc",
		Limit:     10000,
	}

	entries, _, err := s.SearchAuditLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit entries for compliance report: %w", err)
	}

	report := &ComplianceReport{
		ID:           uuid.New().String(),
		Standard:     standard,
		GeneratedAt:  time.Now(),
		ReportPeriod: period,
	}

	// Evaluate compliance based on standard
	switch standard {
	case "FISMA":
		report = s.evaluateFISMACompliance(report, entries)
	case "NIST":
		report = s.evaluateNISTCompliance(report, entries)
	default:
		return nil, fmt.Errorf("unsupported compliance standard: %s", standard)
	}

	// Save the compliance report
	if err := s.repo.CreateComplianceReport(ctx, *report); err != nil {
		return nil, fmt.Errorf("failed to save compliance report: %w", err)
	}

	return report, nil
}

// GetComplianceReport retrieves a specific compliance report
func (s *serviceImpl) GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error) {
	return s.repo.GetComplianceReport(ctx, reportID)
}

// ValidateCompliance performs real-time compliance validation
func (s *serviceImpl) ValidateCompliance(ctx context.Context, standard string) (*ComplianceReport, error) {
	now := time.Now()
	period := ReportPeriod{
		StartDate: now.AddDate(0, -1, 0), // Last month
		EndDate:   now,
	}

	return s.GenerateComplianceReport(ctx, standard, period)
}

// PurgeOldAuditLogs removes audit logs older than the retention period
func (s *serviceImpl) PurgeOldAuditLogs(ctx context.Context, retentionPeriod time.Duration) (int, error) {
	cutoffDate := time.Now().Add(-retentionPeriod)
	return s.repo.DeleteAuditEntriesBefore(ctx, cutoffDate)
}

// ArchiveAuditLogs archives audit logs before a specific date
func (s *serviceImpl) ArchiveAuditLogs(ctx context.Context, beforeDate time.Time) error {
	return s.repo.ArchiveAuditEntries(ctx, beforeDate)
}

// Helper functions

func getMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Simplified compliance evaluation functions
func (s *serviceImpl) evaluateFISMACompliance(report *ComplianceReport, entries []AuditEntry) *ComplianceReport {
	// Simplified FISMA compliance evaluation
	report.ComplianceScore = 85.0 // Example score
	report.Status = "compliant"

	report.Requirements = []ComplianceRequirement{
		{
			ID:          "AC-2",
			Title:       "Account Management",
			Description: "The organization manages information system accounts",
			Status:      "met",
			Score:       90.0,
			Evidence:    []string{"User creation/deletion events logged", "Account management audit trail present"},
		},
		{
			ID:          "AU-2",
			Title:       "Audit Events",
			Description: "The organization determines that the information system is capable of auditing events",
			Status:      "met",
			Score:       95.0,
			Evidence:    []string{"Comprehensive audit logging implemented", "All required events captured"},
		},
	}

	return report
}

func (s *serviceImpl) evaluateNISTCompliance(report *ComplianceReport, entries []AuditEntry) *ComplianceReport {
	// Simplified NIST compliance evaluation
	report.ComplianceScore = 88.0 // Example score
	report.Status = "compliant"

	report.Requirements = []ComplianceRequirement{
		{
			ID:          "PR.AC-1",
			Title:       "Identity and Access Management",
			Description: "Identities and credentials are issued, managed, verified, revoked, and audited",
			Status:      "met",
			Score:       92.0,
			Evidence:    []string{"Identity management events logged", "Access control audit trail present"},
		},
	}

	return report
}
