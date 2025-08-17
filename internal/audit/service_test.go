package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateAuditEntry(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepository) GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*AuditEntry), args.Error(1)
}

func (m *MockRepository) SearchAuditEntries(ctx context.Context, query AuditQuery) ([]AuditEntry, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]AuditEntry), args.Int(1), args.Error(2)
}

func (m *MockRepository) DeleteAuditEntriesBefore(ctx context.Context, beforeDate time.Time) (int, error) {
	args := m.Called(ctx, beforeDate)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) ArchiveAuditEntries(ctx context.Context, beforeDate time.Time) error {
	args := m.Called(ctx, beforeDate)
	return args.Error(0)
}

func (m *MockRepository) CreateDataLineageEntry(ctx context.Context, entry DataLineageEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepository) GetDataLineage(ctx context.Context, dataID string) ([]DataLineageEntry, error) {
	args := m.Called(ctx, dataID)
	return args.Get(0).([]DataLineageEntry), args.Error(1)
}

func (m *MockRepository) GetDataProvenance(ctx context.Context, dataID string) (*DataLineageEntry, error) {
	args := m.Called(ctx, dataID)
	return args.Get(0).(*DataLineageEntry), args.Error(1)
}

func (m *MockRepository) CreateAuditReport(ctx context.Context, report AuditReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockRepository) GetAuditReport(ctx context.Context, reportID string) (*AuditReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*AuditReport), args.Error(1)
}

func (m *MockRepository) ListAuditReports(ctx context.Context, limit, offset int) ([]AuditReport, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]AuditReport), args.Error(1)
}

func (m *MockRepository) CreateSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]SecurityAlert, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]SecurityAlert), args.Error(1)
}

func (m *MockRepository) UpdateSecurityAlert(ctx context.Context, alertID string, updates map[string]interface{}) error {
	args := m.Called(ctx, alertID, updates)
	return args.Error(0)
}

func (m *MockRepository) CreateComplianceReport(ctx context.Context, report ComplianceReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockRepository) GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*ComplianceReport), args.Error(1)
}

func TestService_LogEvent(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	entry := AuditEntry{
		EventType: EventUserLogin,
		Level:     AuditLevelInfo,
		UserID:    stringPtr("user123"),
		Resource:  "/auth/login",
		Action:    "login",
		Result:    "success",
	}

	mockRepo.On("CreateAuditEntry", ctx, mock.MatchedBy(func(e AuditEntry) bool {
		return e.EventType == EventUserLogin && e.Level == AuditLevelInfo
	})).Return(nil)

	err := service.LogEvent(ctx, entry)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_LogUserAction(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	// Create context with request information
	ctx := context.WithValue(context.Background(), "request_id", "req123")
	ctx = context.WithValue(ctx, "ip_address", "192.168.1.1")
	ctx = context.WithValue(ctx, "user_agent", "test-agent")

	mockRepo.On("CreateAuditEntry", ctx, mock.MatchedBy(func(e AuditEntry) bool {
		return e.UserID != nil && *e.UserID == "user123" &&
			e.Action == "document_upload" &&
			e.Resource == "/documents/123" &&
			e.IPAddress == "192.168.1.1"
	})).Return(nil)

	err := service.LogUserAction(ctx, "user123", "document_upload", "/documents/123", map[string]interface{}{
		"document_id": "doc123",
		"size":        1024,
	})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_LogSecurityEvent(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	userID := "user123"

	// Expect both audit entry and security alert creation for critical events
	mockRepo.On("CreateAuditEntry", ctx, mock.MatchedBy(func(e AuditEntry) bool {
		return e.EventType == EventSecurityViolation && e.Level == AuditLevelSecurity
	})).Return(nil)

	mockRepo.On("CreateSecurityAlert", ctx, mock.MatchedBy(func(a SecurityAlert) bool {
		return a.AlertType == string(EventSecurityViolation) && a.Severity == AuditLevelCritical
	})).Return(nil)

	err := service.LogSecurityEvent(ctx, EventSecurityViolation, &userID, "192.168.1.1", "/sensitive/data", map[string]interface{}{
		"violation_type": "unauthorized_access",
	})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_SearchAuditLogs(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	query := AuditQuery{
		StartDate:  &startDate,
		EndDate:    &endDate,
		EventTypes: []AuditEventType{EventUserLogin, EventUserLogout},
		Limit:      100,
		Offset:     0,
	}

	expectedEntries := []AuditEntry{
		{
			ID:        "entry1",
			EventType: EventUserLogin,
			Level:     AuditLevelInfo,
			UserID:    stringPtr("user123"),
		},
		{
			ID:        "entry2",
			EventType: EventUserLogout,
			Level:     AuditLevelInfo,
			UserID:    stringPtr("user123"),
		},
	}

	mockRepo.On("SearchAuditEntries", ctx, query).Return(expectedEntries, 2, nil)

	entries, total, err := service.SearchAuditLogs(ctx, query)

	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, entries, 2)
	assert.Equal(t, EventUserLogin, entries[0].EventType)
	assert.Equal(t, EventUserLogout, entries[1].EventType)
	mockRepo.AssertExpectations(t)
}

func TestService_TrackDataLineage(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	entry := DataLineageEntry{
		DataID:         "doc123",
		DataType:       "document",
		SourceID:       stringPtr("upload456"),
		SourceType:     stringPtr("file_upload"),
		Transformation: "text_extraction",
		ProcessedBy:    "document_service",
		Version:        1,
		Metadata: map[string]interface{}{
			"original_format": "pdf",
			"extracted_pages": 10,
		},
	}

	mockRepo.On("CreateDataLineageEntry", ctx, mock.MatchedBy(func(e DataLineageEntry) bool {
		return e.DataID == "doc123" && e.DataType == "document"
	})).Return(nil)

	err := service.TrackDataLineage(ctx, entry)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_GenerateAuditReport(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	query := AuditQuery{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     1000,
	}

	entries := []AuditEntry{
		{
			ID:        "entry1",
			EventType: EventUserLogin,
			Level:     AuditLevelInfo,
			Result:    "success",
			UserID:    stringPtr("user1"),
			Resource:  "/auth/login",
		},
		{
			ID:        "entry2",
			EventType: EventDocumentUploaded,
			Level:     AuditLevelInfo,
			Result:    "success",
			UserID:    stringPtr("user2"),
			Resource:  "/documents",
		},
		{
			ID:        "entry3",
			EventType: EventUserLoginFailed,
			Level:     AuditLevelWarning,
			Result:    "failure",
			UserID:    stringPtr("user3"),
			Resource:  "/auth/login",
		},
	}

	mockRepo.On("SearchAuditEntries", ctx, query).Return(entries, 3, nil)
	mockRepo.On("CreateAuditReport", ctx, mock.MatchedBy(func(r AuditReport) bool {
		return r.Summary.TotalEvents == 3 &&
			r.Summary.EventsByType[EventUserLogin] == 1 &&
			r.Summary.EventsByType[EventDocumentUploaded] == 1 &&
			r.Summary.EventsByType[EventUserLoginFailed] == 1 &&
			r.Summary.FailedOperations == 1 &&
			r.Summary.UniqueUsers == 3 &&
			r.Summary.UniqueResources == 2
	})).Return(nil)

	report, err := service.GenerateAuditReport(ctx, query, "json")

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 3, report.Summary.TotalEvents)
	assert.Equal(t, 1, report.Summary.EventsByType[EventUserLogin])
	assert.Equal(t, 1, report.Summary.EventsByType[EventDocumentUploaded])
	assert.Equal(t, 1, report.Summary.EventsByType[EventUserLoginFailed])
	assert.Equal(t, 1, report.Summary.FailedOperations)
	assert.Equal(t, 3, report.Summary.UniqueUsers)
	assert.Equal(t, 2, report.Summary.UniqueResources)
	mockRepo.AssertExpectations(t)
}

func TestService_DetectAnomalies(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	userID := "user123"
	timeWindow := 5 * time.Minute

	// Mock user activity with multiple failed logins
	entries := []AuditEntry{
		{EventType: EventUserLoginFailed, Result: "failure", IPAddress: "192.168.1.1"},
		{EventType: EventUserLoginFailed, Result: "failure", IPAddress: "192.168.1.1"},
		{EventType: EventUserLoginFailed, Result: "failure", IPAddress: "192.168.1.1"},
		{EventType: EventUserLoginFailed, Result: "failure", IPAddress: "192.168.1.1"},
		{EventType: EventUserLoginFailed, Result: "failure", IPAddress: "192.168.1.1"},
		{EventType: EventUserLoginFailed, Result: "failure", IPAddress: "192.168.1.1"},
		{EventType: EventUserLogin, Result: "success", IPAddress: "192.168.1.2"},
		{EventType: EventUserLogin, Result: "success", IPAddress: "10.0.0.1"},
		{EventType: EventUserLogin, Result: "success", IPAddress: "172.16.0.1"},
		{EventType: EventUserLogin, Result: "success", IPAddress: "203.0.113.1"},
	}

	mockRepo.On("SearchAuditEntries", ctx, mock.MatchedBy(func(q AuditQuery) bool {
		return len(q.UserIDs) == 1 && q.UserIDs[0] == userID
	})).Return(entries, len(entries), nil)

	alerts, err := service.DetectAnomalies(ctx, userID, timeWindow)

	assert.NoError(t, err)
	assert.Len(t, alerts, 2) // Should detect both excessive failed logins and multiple IP logins

	// Check for excessive failed logins alert
	foundFailedLoginsAlert := false
	foundMultipleIPAlert := false

	for _, alert := range alerts {
		if alert.AlertType == "EXCESSIVE_FAILED_LOGINS" {
			foundFailedLoginsAlert = true
			assert.Equal(t, AuditLevelWarning, alert.Severity)
			assert.Equal(t, &userID, alert.UserID)
		}
		if alert.AlertType == "MULTIPLE_IP_LOGINS" {
			foundMultipleIPAlert = true
			assert.Equal(t, AuditLevelWarning, alert.Severity)
			assert.Equal(t, &userID, alert.UserID)
		}
	}

	assert.True(t, foundFailedLoginsAlert, "Should detect excessive failed logins")
	assert.True(t, foundMultipleIPAlert, "Should detect multiple IP logins")
	mockRepo.AssertExpectations(t)
}

func TestService_GenerateComplianceReport(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	period := ReportPeriod{
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	entries := []AuditEntry{
		{EventType: EventUserLogin, Level: AuditLevelInfo},
		{EventType: EventDocumentUploaded, Level: AuditLevelInfo},
	}

	mockRepo.On("SearchAuditEntries", ctx, mock.AnythingOfType("AuditQuery")).Return(entries, 2, nil)
	mockRepo.On("CreateComplianceReport", ctx, mock.MatchedBy(func(r ComplianceReport) bool {
		return r.Standard == "FISMA" && r.ComplianceScore > 0
	})).Return(nil)

	report, err := service.GenerateComplianceReport(ctx, "FISMA", period)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "FISMA", report.Standard)
	assert.Greater(t, report.ComplianceScore, 0.0)
	assert.NotEmpty(t, report.Requirements)
	mockRepo.AssertExpectations(t)
}

func TestService_PurgeOldAuditLogs(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	retentionPeriod := 365 * 24 * time.Hour // 1 year

	mockRepo.On("DeleteAuditEntriesBefore", ctx, mock.MatchedBy(func(date time.Time) bool {
		return date.Before(time.Now())
	})).Return(150, nil)

	deletedCount, err := service.PurgeOldAuditLogs(ctx, retentionPeriod)

	assert.NoError(t, err)
	assert.Equal(t, 150, deletedCount)
	mockRepo.AssertExpectations(t)
}
