package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Helper function
func stringPtr(s string) *string {
	return &s
}

func setupTestDB(t *testing.T) *mongo.Database {
	// Connect to test MongoDB instance with authentication
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://admin:password@localhost:27017"))
	require.NoError(t, err)

	// Use a test database
	db := client.Database("audit_test")

	// Clean up any existing test data
	db.Drop(context.Background())

	return db
}

func TestMongoRepository_CreateAndGetAuditEntry(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	entry := AuditEntry{
		ID:        "test-entry-1",
		Timestamp: time.Now(),
		EventType: EventUserLogin,
		Level:     AuditLevelInfo,
		UserID:    stringPtr("user123"),
		IPAddress: "192.168.1.1",
		UserAgent: "test-agent",
		Resource:  "/auth/login",
		Action:    "login",
		Result:    "success",
		RequestID: "req123",
		Details: map[string]interface{}{
			"method": "POST",
			"status": 200,
		},
	}

	// Create audit entry
	err := repo.CreateAuditEntry(ctx, entry)
	assert.NoError(t, err)

	// Retrieve audit entry
	retrieved, err := repo.GetAuditEntry(ctx, entry.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, entry.ID, retrieved.ID)
	assert.Equal(t, entry.EventType, retrieved.EventType)
	assert.Equal(t, entry.Level, retrieved.Level)
	assert.Equal(t, entry.UserID, retrieved.UserID)
	assert.Equal(t, entry.Resource, retrieved.Resource)
}

func TestMongoRepository_SearchAuditEntries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	// Create test entries
	now := time.Now()
	entries := []AuditEntry{
		{
			ID:        "entry1",
			Timestamp: now.Add(-2 * time.Hour),
			EventType: EventUserLogin,
			Level:     AuditLevelInfo,
			UserID:    stringPtr("user1"),
			Resource:  "/auth/login",
			Result:    "success",
		},
		{
			ID:        "entry2",
			Timestamp: now.Add(-1 * time.Hour),
			EventType: EventDocumentUploaded,
			Level:     AuditLevelInfo,
			UserID:    stringPtr("user2"),
			Resource:  "/documents",
			Result:    "success",
		},
		{
			ID:        "entry3",
			Timestamp: now,
			EventType: EventUserLoginFailed,
			Level:     AuditLevelWarning,
			UserID:    stringPtr("user1"),
			Resource:  "/auth/login",
			Result:    "failure",
		},
	}

	for _, entry := range entries {
		err := repo.CreateAuditEntry(ctx, entry)
		require.NoError(t, err)
	}

	// Test search by user ID
	query := AuditQuery{
		UserIDs:   []string{"user1"},
		SortBy:    "timestamp",
		SortOrder: "desc",
		Limit:     10,
	}

	results, total, err := repo.SearchAuditEntries(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
	assert.Equal(t, "entry3", results[0].ID) // Most recent first
	assert.Equal(t, "entry1", results[1].ID)

	// Test search by event type
	query = AuditQuery{
		EventTypes: []AuditEventType{EventUserLogin, EventUserLoginFailed},
		SortBy:     "timestamp",
		SortOrder:  "asc",
		Limit:      10,
	}

	results, total, err = repo.SearchAuditEntries(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
	assert.Equal(t, "entry1", results[0].ID) // Oldest first
	assert.Equal(t, "entry3", results[1].ID)

	// Test search by date range
	startDate := now.Add(-90 * time.Minute)
	endDate := now.Add(-30 * time.Minute)
	query = AuditQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     10,
	}

	results, total, err = repo.SearchAuditEntries(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "entry2", results[0].ID)

	// Test pagination
	query = AuditQuery{
		Limit:     1,
		Offset:    1,
		SortBy:    "timestamp",
		SortOrder: "asc",
	}

	results, total, err = repo.SearchAuditEntries(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, results, 1)
	assert.Equal(t, "entry2", results[0].ID) // Second entry when sorted by timestamp asc
}

func TestMongoRepository_DataLineage(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	// Create data lineage entries
	entries := []DataLineageEntry{
		{
			ID:             "lineage1",
			DataID:         "doc123",
			DataType:       "document",
			SourceID:       stringPtr("upload456"),
			SourceType:     stringPtr("file_upload"),
			Transformation: "text_extraction",
			ProcessedBy:    "document_service",
			ProcessedAt:    time.Now().Add(-2 * time.Hour),
			Version:        1,
		},
		{
			ID:             "lineage2",
			DataID:         "doc123",
			DataType:       "document",
			SourceID:       stringPtr("lineage1"),
			SourceType:     stringPtr("processed_document"),
			Transformation: "embedding_generation",
			ProcessedBy:    "embedding_service",
			ProcessedAt:    time.Now().Add(-1 * time.Hour),
			Version:        2,
		},
	}

	for _, entry := range entries {
		err := repo.CreateDataLineageEntry(ctx, entry)
		require.NoError(t, err)
	}

	// Get complete lineage
	lineage, err := repo.GetDataLineage(ctx, "doc123")
	assert.NoError(t, err)
	assert.Len(t, lineage, 2)
	assert.Equal(t, "lineage2", lineage[0].ID) // Most recent first
	assert.Equal(t, "lineage1", lineage[1].ID)

	// Get data provenance (most recent entry)
	provenance, err := repo.GetDataProvenance(ctx, "doc123")
	assert.NoError(t, err)
	assert.NotNil(t, provenance)
	assert.Equal(t, "lineage2", provenance.ID)
	assert.Equal(t, "embedding_generation", provenance.Transformation)
}

func TestMongoRepository_SecurityAlerts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	alert := SecurityAlert{
		ID:          "alert1",
		Timestamp:   time.Now(),
		AlertType:   "EXCESSIVE_FAILED_LOGINS",
		Severity:    AuditLevelWarning,
		Title:       "Too many failed login attempts",
		Description: "User has exceeded failed login threshold",
		UserID:      stringPtr("user123"),
		IPAddress:   "192.168.1.1",
		Resource:    "/auth/login",
		Status:      "open",
		Details: map[string]interface{}{
			"failed_attempts": 6,
			"time_window":     "5m",
		},
	}

	// Create security alert
	err := repo.CreateSecurityAlert(ctx, alert)
	assert.NoError(t, err)

	// Get security alerts
	alerts, err := repo.GetSecurityAlerts(ctx, "open", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, alert.ID, alerts[0].ID)
	assert.Equal(t, alert.AlertType, alerts[0].AlertType)

	// Update security alert
	updates := map[string]interface{}{
		"status":     "resolved",
		"resolution": "False positive - legitimate user",
	}

	err = repo.UpdateSecurityAlert(ctx, alert.ID, updates)
	assert.NoError(t, err)

	// Verify update
	alerts, err = repo.GetSecurityAlerts(ctx, "resolved", 10, 0)
	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
	assert.Equal(t, "resolved", alerts[0].Status)
	assert.Equal(t, "False positive - legitimate user", *alerts[0].Resolution)
}

func TestMongoRepository_AuditReports(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	report := AuditReport{
		ID:          "report1",
		Title:       "Weekly Audit Report",
		Description: "Audit report for the past week",
		GeneratedBy: "admin",
		GeneratedAt: time.Now(),
		StartDate:   time.Now().AddDate(0, 0, -7),
		EndDate:     time.Now(),
		Format:      "json",
		Summary: ReportSummary{
			TotalEvents:      100,
			EventsByType:     map[AuditEventType]int{EventUserLogin: 50, EventDocumentUploaded: 30},
			EventsByLevel:    map[AuditLevel]int{AuditLevelInfo: 90, AuditLevelWarning: 10},
			UniqueUsers:      25,
			UniqueResources:  10,
			SecurityEvents:   5,
			FailedOperations: 8,
		},
		Entries: []AuditEntry{
			{
				ID:        "entry1",
				EventType: EventUserLogin,
				Level:     AuditLevelInfo,
			},
		},
	}

	// Create audit report
	err := repo.CreateAuditReport(ctx, report)
	assert.NoError(t, err)

	// Get audit report
	retrieved, err := repo.GetAuditReport(ctx, report.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, report.ID, retrieved.ID)
	assert.Equal(t, report.Title, retrieved.Title)
	assert.Equal(t, report.Summary.TotalEvents, retrieved.Summary.TotalEvents)

	// List audit reports
	reports, err := repo.ListAuditReports(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, reports, 1)
	assert.Equal(t, report.ID, reports[0].ID)
}

func TestMongoRepository_ComplianceReports(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	report := ComplianceReport{
		ID:              "compliance1",
		Standard:        "FISMA",
		GeneratedBy:     "compliance_officer",
		GeneratedAt:     time.Now(),
		ComplianceScore: 85.5,
		Status:          "compliant",
		ReportPeriod: ReportPeriod{
			StartDate: time.Now().AddDate(0, -1, 0),
			EndDate:   time.Now(),
		},
		Requirements: []ComplianceRequirement{
			{
				ID:          "AC-2",
				Title:       "Account Management",
				Description: "The organization manages information system accounts",
				Status:      "met",
				Score:       90.0,
				Evidence:    []string{"User management audit logs", "Account lifecycle documentation"},
			},
		},
		Findings: []ComplianceFinding{
			{
				ID:          "finding1",
				Severity:    AuditLevelWarning,
				Title:       "Incomplete audit logging",
				Description: "Some API endpoints lack comprehensive audit logging",
				Requirement: "AU-2",
				Remediation: "Implement audit logging for all API endpoints",
			},
		},
	}

	// Create compliance report
	err := repo.CreateComplianceReport(ctx, report)
	assert.NoError(t, err)

	// Get compliance report
	retrieved, err := repo.GetComplianceReport(ctx, report.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, report.ID, retrieved.ID)
	assert.Equal(t, report.Standard, retrieved.Standard)
	assert.Equal(t, report.ComplianceScore, retrieved.ComplianceScore)
	assert.Len(t, retrieved.Requirements, 1)
	assert.Len(t, retrieved.Findings, 1)
}

func TestMongoRepository_DeleteOldAuditEntries(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMongoRepository(db)
	ctx := context.Background()

	// Create test entries with different timestamps
	now := time.Now()
	entries := []AuditEntry{
		{
			ID:        "old1",
			Timestamp: now.AddDate(0, 0, -10), // 10 days ago
			EventType: EventUserLogin,
			Level:     AuditLevelInfo,
		},
		{
			ID:        "old2",
			Timestamp: now.AddDate(0, 0, -8), // 8 days ago
			EventType: EventUserLogin,
			Level:     AuditLevelInfo,
		},
		{
			ID:        "recent1",
			Timestamp: now.AddDate(0, 0, -2), // 2 days ago
			EventType: EventUserLogin,
			Level:     AuditLevelInfo,
		},
	}

	for _, entry := range entries {
		err := repo.CreateAuditEntry(ctx, entry)
		require.NoError(t, err)
	}

	// Delete entries older than 7 days
	cutoffDate := now.AddDate(0, 0, -7)
	deletedCount, err := repo.DeleteAuditEntriesBefore(ctx, cutoffDate)
	assert.NoError(t, err)
	assert.Equal(t, 2, deletedCount)

	// Verify only recent entry remains
	query := AuditQuery{Limit: 10}
	remaining, total, err := repo.SearchAuditEntries(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "recent1", remaining[0].ID)
}
