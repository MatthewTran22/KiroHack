package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"ai-government-consultant/internal/audit"
)

func setupAuditIntegrationTest(t *testing.T) (audit.Service, func()) {
	// Connect to test MongoDB with authentication
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://admin:password@localhost:27017"))
	require.NoError(t, err)

	// Use test database
	db := client.Database("audit_integration_test")

	// Clean up existing data
	db.Drop(context.Background())

	// Create repository and service
	repo := audit.NewMongoRepository(db)
	service := audit.NewService(repo)

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return service, cleanup
}

func TestAuditIntegration_CompleteWorkflow(t *testing.T) {
	service, cleanup := setupAuditIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test 1: Log various types of events
	t.Run("LogEvents", func(t *testing.T) {
		// Log user login
		err := service.LogUserAction(ctx, "user123", "login", "/auth/login", map[string]interface{}{
			"method":  "POST",
			"success": true,
		})
		assert.NoError(t, err)

		// Log document upload
		err = service.LogUserAction(ctx, "user123", "document_upload", "/documents", map[string]interface{}{
			"document_id": "doc456",
			"size":        1024,
		})
		assert.NoError(t, err)

		// Log security event
		err = service.LogSecurityEvent(ctx, audit.EventUnauthorizedAccess, stringPtr("user456"), "192.168.1.100", "/admin/users", map[string]interface{}{
			"attempted_action": "delete_user",
		})
		assert.NoError(t, err)

		// Log system event
		err = service.LogSystemEvent(ctx, audit.EventSystemStartup, map[string]interface{}{
			"version":     "1.0.0",
			"environment": "test",
		})
		assert.NoError(t, err)
	})

	// Test 2: Search and retrieve audit logs
	t.Run("SearchAuditLogs", func(t *testing.T) {
		// Search by user
		query := audit.AuditQuery{
			UserIDs: []string{"user123"},
			Limit:   10,
		}

		entries, total, err := service.SearchAuditLogs(ctx, query)
		assert.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, entries, 2)

		// Search by event type
		query = audit.AuditQuery{
			EventTypes: []audit.AuditEventType{audit.EventUnauthorizedAccess},
			Limit:      10,
		}

		entries, total, err = service.SearchAuditLogs(ctx, query)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, entries, 1)
		assert.Equal(t, audit.EventUnauthorizedAccess, entries[0].EventType)

		// Search by level
		query = audit.AuditQuery{
			Levels: []audit.AuditLevel{audit.AuditLevelSecurity},
			Limit:  10,
		}

		entries, total, err = service.SearchAuditLogs(ctx, query)
		assert.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, entries, 1)
	})

	// Test 3: Data lineage tracking
	t.Run("DataLineage", func(t *testing.T) {
		// Track document processing lineage
		lineageEntry := audit.DataLineageEntry{
			DataID:         "doc456",
			DataType:       "document",
			SourceID:       stringPtr("upload789"),
			SourceType:     stringPtr("file_upload"),
			Transformation: "text_extraction",
			ProcessedBy:    "document_service",
			Version:        1,
			Metadata: map[string]interface{}{
				"original_format": "pdf",
				"pages":           10,
			},
		}

		err := service.TrackDataLineage(ctx, lineageEntry)
		assert.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		// Track embedding generation
		embeddingEntry := audit.DataLineageEntry{
			DataID:         "doc456",
			DataType:       "document",
			SourceID:       stringPtr("doc456"),
			SourceType:     stringPtr("processed_document"),
			Transformation: "embedding_generation",
			ProcessedBy:    "embedding_service",
			Version:        2,
			Metadata: map[string]interface{}{
				"embedding_model": "text-embedding-ada-002",
				"dimensions":      1536,
			},
		}

		err = service.TrackDataLineage(ctx, embeddingEntry)
		assert.NoError(t, err)

		// Get complete lineage
		lineage, err := service.GetDataLineage(ctx, "doc456")
		assert.NoError(t, err)
		assert.Len(t, lineage, 2)

		// Get data provenance
		provenance, err := service.GetDataProvenance(ctx, "doc456")
		assert.NoError(t, err)
		assert.NotNil(t, provenance)
		assert.Equal(t, "embedding_generation", provenance.Transformation)
	})

	// Test 4: Generate audit report
	t.Run("GenerateAuditReport", func(t *testing.T) {
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		query := audit.AuditQuery{
			StartDate: &startDate,
			EndDate:   &endDate,
			Limit:     1000,
		}

		report, err := service.GenerateAuditReport(ctx, query, "json")
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Greater(t, report.Summary.TotalEvents, 0)
		assert.Greater(t, report.Summary.UniqueUsers, 0)
		assert.NotEmpty(t, report.ID)

		// Retrieve the report
		retrieved, err := service.GetAuditReport(ctx, report.ID)
		assert.NoError(t, err)
		assert.Equal(t, report.ID, retrieved.ID)
		assert.Equal(t, report.Summary.TotalEvents, retrieved.Summary.TotalEvents)
	})

	// Test 5: Security alerts and anomaly detection
	t.Run("SecurityAlertsAndAnomalies", func(t *testing.T) {
		// Create multiple failed login attempts to trigger anomaly detection
		for i := 0; i < 7; i++ {
			err := service.LogSecurityEvent(ctx, audit.EventUserLoginFailed, stringPtr("user789"), "192.168.1.200", "/auth/login", map[string]interface{}{
				"attempt": i + 1,
			})
			assert.NoError(t, err)
		}

		// Detect anomalies
		alerts, err := service.DetectAnomalies(ctx, "user789", 10*time.Minute)
		assert.NoError(t, err)
		assert.NotEmpty(t, alerts)

		// Should detect excessive failed logins
		foundExcessiveFailedLogins := false
		for _, alert := range alerts {
			if alert.AlertType == "EXCESSIVE_FAILED_LOGINS" {
				foundExcessiveFailedLogins = true
				assert.Equal(t, audit.AuditLevelWarning, alert.Severity)
				assert.Equal(t, "user789", *alert.UserID)
			}
		}
		assert.True(t, foundExcessiveFailedLogins)

		// Get security alerts
		retrievedAlerts, err := service.GetSecurityAlerts(ctx, "open", 10, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, retrievedAlerts)

		// Update alert status
		if len(retrievedAlerts) > 0 {
			alertID := retrievedAlerts[0].ID
			resolution := "Investigated - legitimate user with forgotten password"
			err = service.UpdateSecurityAlert(ctx, alertID, "resolved", &resolution)
			assert.NoError(t, err)

			// Verify update
			resolvedAlerts, err := service.GetSecurityAlerts(ctx, "resolved", 10, 0)
			assert.NoError(t, err)
			assert.NotEmpty(t, resolvedAlerts)
		}
	})

	// Test 6: Compliance reporting
	t.Run("ComplianceReporting", func(t *testing.T) {
		period := audit.ReportPeriod{
			StartDate: time.Now().Add(-30 * 24 * time.Hour),
			EndDate:   time.Now(),
		}

		// Generate FISMA compliance report
		report, err := service.GenerateComplianceReport(ctx, "FISMA", period)
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, "FISMA", report.Standard)
		assert.Greater(t, report.ComplianceScore, 0.0)
		assert.NotEmpty(t, report.Requirements)

		// Retrieve compliance report
		retrieved, err := service.GetComplianceReport(ctx, report.ID)
		assert.NoError(t, err)
		assert.Equal(t, report.ID, retrieved.ID)
		assert.Equal(t, report.Standard, retrieved.Standard)

		// Validate compliance (real-time check)
		validation, err := service.ValidateCompliance(ctx, "FISMA")
		assert.NoError(t, err)
		assert.NotNil(t, validation)
		assert.Equal(t, "FISMA", validation.Standard)
	})

	// Test 7: Data retention and cleanup
	t.Run("DataRetentionAndCleanup", func(t *testing.T) {
		// Create old audit entries
		oldEntry := audit.AuditEntry{
			ID:        "old-entry-1",
			Timestamp: time.Now().Add(-400 * 24 * time.Hour), // Over a year old
			EventType: audit.EventUserLogin,
			Level:     audit.AuditLevelInfo,
			UserID:    stringPtr("old-user"),
			Resource:  "/auth/login",
			Action:    "login",
			Result:    "success",
		}

		err := service.LogEvent(ctx, oldEntry)
		assert.NoError(t, err)

		// Purge old logs (retention period of 1 year)
		retentionPeriod := 365 * 24 * time.Hour
		deletedCount, err := service.PurgeOldAuditLogs(ctx, retentionPeriod)
		assert.NoError(t, err)
		assert.Greater(t, deletedCount, 0)

		// Verify old entry was deleted
		_, err = service.GetAuditEntry(ctx, oldEntry.ID)
		assert.Error(t, err) // Should not be found

		// Archive recent entries
		archiveDate := time.Now().Add(-1 * time.Hour)
		err = service.ArchiveAuditLogs(ctx, archiveDate)
		assert.NoError(t, err)
	})
}

func TestAuditIntegration_ExportFunctionality(t *testing.T) {
	service, cleanup := setupAuditIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create some test data
	for i := 0; i < 5; i++ {
		err := service.LogUserAction(ctx, "export-user", "test_action", "/test/resource", map[string]interface{}{
			"iteration": i,
		})
		assert.NoError(t, err)
	}

	// Generate report
	startDate := time.Now().Add(-1 * time.Hour)
	endDate := time.Now()

	query := audit.AuditQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
		UserIDs:   []string{"export-user"},
		Limit:     100,
	}

	report, err := service.GenerateAuditReport(ctx, query, "json")
	assert.NoError(t, err)
	assert.NotNil(t, report)

	// Test JSON export
	jsonData, err := service.ExportAuditReport(ctx, report.ID, "json")
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	assert.Contains(t, string(jsonData), "export-user")

	// Test CSV export
	csvData, err := service.ExportAuditReport(ctx, report.ID, "csv")
	assert.NoError(t, err)
	assert.NotEmpty(t, csvData)
	assert.Contains(t, string(csvData), "export-user")
	assert.Contains(t, string(csvData), "ID,Timestamp,Event Type") // CSV header
}

func TestAuditIntegration_PerformanceAndScaling(t *testing.T) {
	service, cleanup := setupAuditIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Test bulk logging performance
	t.Run("BulkLogging", func(t *testing.T) {
		start := time.Now()

		// Log 100 events
		for i := 0; i < 100; i++ {
			err := service.LogUserAction(ctx, "perf-user", "bulk_action", "/bulk/resource", map[string]interface{}{
				"batch": i,
			})
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("Logged 100 events in %v", duration)

		// Should complete within reasonable time
		assert.Less(t, duration, 10*time.Second)
	})

	// Test search performance with large dataset
	t.Run("SearchPerformance", func(t *testing.T) {
		query := audit.AuditQuery{
			UserIDs: []string{"perf-user"},
			Limit:   50,
		}

		start := time.Now()
		entries, total, err := service.SearchAuditLogs(ctx, query)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, 100, total)
		assert.Len(t, entries, 50) // Limited to 50

		t.Logf("Searched %d entries in %v", total, duration)

		// Search should be fast
		assert.Less(t, duration, 5*time.Second)
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
