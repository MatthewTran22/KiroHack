package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuditService is a mock implementation of the audit Service interface
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) LogEvent(ctx context.Context, entry AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditService) LogUserAction(ctx context.Context, userID, action, resource string, details map[string]interface{}) error {
	args := m.Called(ctx, userID, action, resource, details)
	return args.Error(0)
}

func (m *MockAuditService) LogSystemEvent(ctx context.Context, eventType AuditEventType, details map[string]interface{}) error {
	args := m.Called(ctx, eventType, details)
	return args.Error(0)
}

func (m *MockAuditService) LogSecurityEvent(ctx context.Context, eventType AuditEventType, userID *string, ipAddress, resource string, details map[string]interface{}) error {
	args := m.Called(ctx, eventType, userID, ipAddress, resource, details)
	return args.Error(0)
}

func (m *MockAuditService) SearchAuditLogs(ctx context.Context, query AuditQuery) ([]AuditEntry, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]AuditEntry), args.Int(1), args.Error(2)
}

func (m *MockAuditService) GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*AuditEntry), args.Error(1)
}

func (m *MockAuditService) GetUserActivity(ctx context.Context, userID string, startDate, endDate time.Time) ([]AuditEntry, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	return args.Get(0).([]AuditEntry), args.Error(1)
}

func (m *MockAuditService) TrackDataLineage(ctx context.Context, entry DataLineageEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditService) GetDataLineage(ctx context.Context, dataID string) ([]DataLineageEntry, error) {
	args := m.Called(ctx, dataID)
	return args.Get(0).([]DataLineageEntry), args.Error(1)
}

func (m *MockAuditService) GetDataProvenance(ctx context.Context, dataID string) (*DataLineageEntry, error) {
	args := m.Called(ctx, dataID)
	return args.Get(0).(*DataLineageEntry), args.Error(1)
}

func (m *MockAuditService) GenerateAuditReport(ctx context.Context, query AuditQuery, format string) (*AuditReport, error) {
	args := m.Called(ctx, query, format)
	return args.Get(0).(*AuditReport), args.Error(1)
}

func (m *MockAuditService) GetAuditReport(ctx context.Context, reportID string) (*AuditReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*AuditReport), args.Error(1)
}

func (m *MockAuditService) ListAuditReports(ctx context.Context, limit, offset int) ([]AuditReport, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]AuditReport), args.Error(1)
}

func (m *MockAuditService) ExportAuditReport(ctx context.Context, reportID, format string) ([]byte, error) {
	args := m.Called(ctx, reportID, format)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockAuditService) CreateSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAuditService) GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]SecurityAlert, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]SecurityAlert), args.Error(1)
}

func (m *MockAuditService) UpdateSecurityAlert(ctx context.Context, alertID, status string, resolution *string) error {
	args := m.Called(ctx, alertID, status, resolution)
	return args.Error(0)
}

func (m *MockAuditService) DetectAnomalies(ctx context.Context, userID string, timeWindow time.Duration) ([]SecurityAlert, error) {
	args := m.Called(ctx, userID, timeWindow)
	return args.Get(0).([]SecurityAlert), args.Error(1)
}

func (m *MockAuditService) GenerateComplianceReport(ctx context.Context, standard string, period ReportPeriod) (*ComplianceReport, error) {
	args := m.Called(ctx, standard, period)
	return args.Get(0).(*ComplianceReport), args.Error(1)
}

func (m *MockAuditService) GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*ComplianceReport), args.Error(1)
}

func (m *MockAuditService) ValidateCompliance(ctx context.Context, standard string) (*ComplianceReport, error) {
	args := m.Called(ctx, standard)
	return args.Get(0).(*ComplianceReport), args.Error(1)
}

func (m *MockAuditService) PurgeOldAuditLogs(ctx context.Context, retentionPeriod time.Duration) (int, error) {
	args := m.Called(ctx, retentionPeriod)
	return args.Int(0), args.Error(1)
}

func (m *MockAuditService) ArchiveAuditLogs(ctx context.Context, beforeDate time.Time) error {
	args := m.Called(ctx, beforeDate)
	return args.Error(0)
}

func TestAuditMiddleware_SuccessfulRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuditService)

	// Expect audit log entry for successful request
	mockService.On("LogEvent", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.Action == "GET" &&
			entry.Resource == "/api/documents" &&
			entry.Result == "success" &&
			entry.Level == AuditLevelInfo &&
			entry.RequestID != ""
	})).Return(nil)

	router := gin.New()
	router.Use(AuditMiddleware(mockService))
	router.GET("/api/documents", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/documents", nil)
	req.Header.Set("User-Agent", "test-client")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockService.AssertExpectations(t)
}

func TestAuditMiddleware_FailedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuditService)

	// Expect audit log entry for failed request
	mockService.On("LogEvent", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.Action == "POST" &&
			entry.Resource == "/api/documents" &&
			entry.Result == "failure" &&
			entry.Level == AuditLevelWarning &&
			entry.Details["status_code"] == 400
	})).Return(nil)

	router := gin.New()
	router.Use(AuditMiddleware(mockService))
	router.POST("/api/documents", func(c *gin.Context) {
		c.JSON(400, gin.H{"error": "bad request"})
	})

	req, _ := http.NewRequest("POST", "/api/documents", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockService.AssertExpectations(t)
}

func TestAuditMiddleware_WithUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuditService)

	// Expect audit log entry with user information
	mockService.On("LogEvent", mock.Anything, mock.MatchedBy(func(entry AuditEntry) bool {
		return entry.UserID != nil &&
			*entry.UserID == "user123" &&
			entry.SessionID != nil &&
			*entry.SessionID == "session456"
	})).Return(nil)

	router := gin.New()

	// Middleware to set user context (simulating auth middleware)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Set("session_id", "session456")
		c.Next()
	})

	router.Use(AuditMiddleware(mockService))
	router.GET("/api/profile", func(c *gin.Context) {
		c.JSON(200, gin.H{"user": "profile"})
	})

	req, _ := http.NewRequest("GET", "/api/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockService.AssertExpectations(t)
}

func TestSecurityEventMiddleware_UnauthorizedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuditService)

	// Expect security event log for unauthorized access
	mockService.On("LogSecurityEvent",
		mock.Anything,
		EventUnauthorizedAccess,
		(*string)(nil),                // No user ID for unauthorized access
		mock.AnythingOfType("string"), // IP address
		"/api/admin",
		mock.MatchedBy(func(details map[string]interface{}) bool {
			return details["method"] == "GET" && details["status"] == 403
		}),
	).Return(nil)

	// Mock anomaly detection (no anomalies for this test)
	mockService.On("DetectAnomalies",
		mock.Anything,
		mock.AnythingOfType("string"),
		5*time.Minute,
	).Return([]SecurityAlert{}, nil)

	router := gin.New()
	router.Use(SecurityEventMiddleware(mockService))
	router.GET("/api/admin", func(c *gin.Context) {
		c.JSON(403, gin.H{"error": "forbidden"})
	})

	req, _ := http.NewRequest("GET", "/api/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockService.AssertExpectations(t)
}

func TestSecurityEventMiddleware_FailedAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuditService)

	// Expect security event log for failed authentication
	mockService.On("LogSecurityEvent",
		mock.Anything,
		EventUserLoginFailed,
		(*string)(nil),                // No user ID for failed auth
		mock.AnythingOfType("string"), // IP address
		"/auth/login",
		mock.MatchedBy(func(details map[string]interface{}) bool {
			return details["method"] == "POST" && details["status"] == 401
		}),
	).Return(nil)

	router := gin.New()
	router.Use(SecurityEventMiddleware(mockService))
	router.POST("/auth/login", func(c *gin.Context) {
		c.JSON(401, gin.H{"error": "invalid credentials"})
	})

	req, _ := http.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockService.AssertExpectations(t)
}

func TestSecurityEventMiddleware_AnomalyDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockAuditService)

	// Mock anomaly detection returning alerts
	alerts := []SecurityAlert{
		{
			ID:        "alert1",
			AlertType: "EXCESSIVE_FAILED_LOGINS",
			Severity:  AuditLevelWarning,
		},
	}

	mockService.On("DetectAnomalies",
		mock.Anything,
		"user123",
		5*time.Minute,
	).Return(alerts, nil)

	// Expect security alert creation
	mockService.On("CreateSecurityAlert",
		mock.Anything,
		mock.MatchedBy(func(alert SecurityAlert) bool {
			return alert.AlertType == "EXCESSIVE_FAILED_LOGINS"
		}),
	).Return(nil)

	router := gin.New()

	// Middleware to set user context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Next()
	})

	router.Use(SecurityEventMiddleware(mockService))
	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": "success"})
	})

	req, _ := http.NewRequest("GET", "/api/data", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Give some time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	mockService.AssertExpectations(t)
}

func TestDetermineEventType(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected AuditEventType
	}{
		{"POST", "/auth/login", EventUserLogin},
		{"POST", "/auth/logout", EventUserLogout},
		{"POST", "/documents", EventDocumentUploaded},
		{"GET", "/documents/123", EventDocumentAccessed},
		{"DELETE", "/documents/123", EventDocumentDeleted},
		{"PUT", "/documents/123", EventDocumentUpdated},
		{"POST", "/consultations", EventConsultationStarted},
		{"GET", "/consultations/456", EventConsultationCompleted},
		{"POST", "/knowledge", EventKnowledgeAdded},
		{"GET", "/knowledge/789", EventKnowledgeAccessed},
		{"PUT", "/knowledge/789", EventKnowledgeUpdated},
		{"DELETE", "/knowledge/789", EventKnowledgeDeleted},
		{"GET", "/api/other", AuditEventType("API_REQUEST")},
	}

	for _, test := range tests {
		result := determineEventType(test.method, test.path)
		assert.Equal(t, test.expected, result, "Method: %s, Path: %s", test.method, test.path)
	}
}

func TestDetermineLevelFromStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected AuditLevel
	}{
		{200, AuditLevelInfo},
		{201, AuditLevelInfo},
		{301, AuditLevelInfo},
		{400, AuditLevelWarning},
		{401, AuditLevelWarning},
		{403, AuditLevelWarning},
		{404, AuditLevelWarning},
		{500, AuditLevelError},
		{502, AuditLevelError},
		{503, AuditLevelError},
	}

	for _, test := range tests {
		result := determineLevelFromStatus(test.status)
		assert.Equal(t, test.expected, result, "Status: %d", test.status)
	}
}
