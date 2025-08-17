package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ai-government-consultant/internal/audit"
)

// MockAuditService is a mock implementation of the audit Service interface
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) LogEvent(ctx context.Context, entry audit.AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditService) LogUserAction(ctx context.Context, userID, action, resource string, details map[string]interface{}) error {
	args := m.Called(ctx, userID, action, resource, details)
	return args.Error(0)
}

func (m *MockAuditService) LogSystemEvent(ctx context.Context, eventType audit.AuditEventType, details map[string]interface{}) error {
	args := m.Called(ctx, eventType, details)
	return args.Error(0)
}

func (m *MockAuditService) LogSecurityEvent(ctx context.Context, eventType audit.AuditEventType, userID *string, ipAddress, resource string, details map[string]interface{}) error {
	args := m.Called(ctx, eventType, userID, ipAddress, resource, details)
	return args.Error(0)
}

func (m *MockAuditService) SearchAuditLogs(ctx context.Context, query audit.AuditQuery) ([]audit.AuditEntry, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]audit.AuditEntry), args.Int(1), args.Error(2)
}

func (m *MockAuditService) GetAuditEntry(ctx context.Context, id string) (*audit.AuditEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*audit.AuditEntry), args.Error(1)
}

func (m *MockAuditService) GetUserActivity(ctx context.Context, userID string, startDate, endDate time.Time) ([]audit.AuditEntry, error) {
	args := m.Called(ctx, userID, startDate, endDate)
	return args.Get(0).([]audit.AuditEntry), args.Error(1)
}

func (m *MockAuditService) TrackDataLineage(ctx context.Context, entry audit.DataLineageEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockAuditService) GetDataLineage(ctx context.Context, dataID string) ([]audit.DataLineageEntry, error) {
	args := m.Called(ctx, dataID)
	return args.Get(0).([]audit.DataLineageEntry), args.Error(1)
}

func (m *MockAuditService) GetDataProvenance(ctx context.Context, dataID string) (*audit.DataLineageEntry, error) {
	args := m.Called(ctx, dataID)
	return args.Get(0).(*audit.DataLineageEntry), args.Error(1)
}

func (m *MockAuditService) GenerateAuditReport(ctx context.Context, query audit.AuditQuery, format string) (*audit.AuditReport, error) {
	args := m.Called(ctx, query, format)
	return args.Get(0).(*audit.AuditReport), args.Error(1)
}

func (m *MockAuditService) GetAuditReport(ctx context.Context, reportID string) (*audit.AuditReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*audit.AuditReport), args.Error(1)
}

func (m *MockAuditService) ListAuditReports(ctx context.Context, limit, offset int) ([]audit.AuditReport, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]audit.AuditReport), args.Error(1)
}

func (m *MockAuditService) ExportAuditReport(ctx context.Context, reportID, format string) ([]byte, error) {
	args := m.Called(ctx, reportID, format)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockAuditService) CreateSecurityAlert(ctx context.Context, alert audit.SecurityAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAuditService) GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]audit.SecurityAlert, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]audit.SecurityAlert), args.Error(1)
}

func (m *MockAuditService) UpdateSecurityAlert(ctx context.Context, alertID, status string, resolution *string) error {
	args := m.Called(ctx, alertID, status, resolution)
	return args.Error(0)
}

func (m *MockAuditService) DetectAnomalies(ctx context.Context, userID string, timeWindow time.Duration) ([]audit.SecurityAlert, error) {
	args := m.Called(ctx, userID, timeWindow)
	return args.Get(0).([]audit.SecurityAlert), args.Error(1)
}

func (m *MockAuditService) GenerateComplianceReport(ctx context.Context, standard string, period audit.ReportPeriod) (*audit.ComplianceReport, error) {
	args := m.Called(ctx, standard, period)
	return args.Get(0).(*audit.ComplianceReport), args.Error(1)
}

func (m *MockAuditService) GetComplianceReport(ctx context.Context, reportID string) (*audit.ComplianceReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*audit.ComplianceReport), args.Error(1)
}

func (m *MockAuditService) ValidateCompliance(ctx context.Context, standard string) (*audit.ComplianceReport, error) {
	args := m.Called(ctx, standard)
	return args.Get(0).(*audit.ComplianceReport), args.Error(1)
}

func (m *MockAuditService) PurgeOldAuditLogs(ctx context.Context, retentionPeriod time.Duration) (int, error) {
	args := m.Called(ctx, retentionPeriod)
	return args.Int(0), args.Error(1)
}

func (m *MockAuditService) ArchiveAuditLogs(ctx context.Context, beforeDate time.Time) error {
	args := m.Called(ctx, beforeDate)
	return args.Error(0)
}

func setupAuditRouter(mockService audit.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuditHandler(mockService)

	// Audit logs routes
	router.GET("/api/audit/logs", handler.SearchAuditLogs)
	router.GET("/api/audit/logs/:id", handler.GetAuditEntry)
	router.GET("/api/audit/users/:user_id/activity", handler.GetUserActivity)

	// Data lineage routes
	router.GET("/api/audit/lineage/:data_id", handler.GetDataLineage)
	router.GET("/api/audit/provenance/:data_id", handler.GetDataProvenance)

	// Audit reports routes
	router.POST("/api/audit/reports", handler.GenerateAuditReport)
	router.GET("/api/audit/reports", handler.ListAuditReports)
	router.GET("/api/audit/reports/:id", handler.GetAuditReport)
	router.GET("/api/audit/reports/:id/export", handler.ExportAuditReport)

	// Security alerts routes
	router.GET("/api/audit/security/alerts", handler.GetSecurityAlerts)
	router.PUT("/api/audit/security/alerts/:id", handler.UpdateSecurityAlert)
	router.POST("/api/audit/security/anomalies", handler.DetectAnomalies)

	// Compliance routes
	router.POST("/api/audit/compliance/reports", handler.GenerateComplianceReport)
	router.GET("/api/audit/compliance/reports/:id", handler.GetComplianceReport)
	router.GET("/api/audit/compliance/validate/:standard", handler.ValidateCompliance)

	return router
}

func TestAuditHandler_SearchAuditLogs(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	entries := []audit.AuditEntry{
		{
			ID:        "entry1",
			EventType: audit.EventUserLogin,
			Level:     audit.AuditLevelInfo,
			UserID:    stringPtr("user123"),
		},
		{
			ID:        "entry2",
			EventType: audit.EventDocumentUploaded,
			Level:     audit.AuditLevelInfo,
			UserID:    stringPtr("user456"),
		},
	}

	mockService.On("SearchAuditLogs", mock.Anything, mock.MatchedBy(func(q audit.AuditQuery) bool {
		return len(q.UserIDs) == 1 && q.UserIDs[0] == "user123" && q.Limit == 50
	})).Return(entries, 2, nil)

	req, _ := http.NewRequest("GET", "/api/audit/logs?user_ids=user123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), response["total"])
	assert.Equal(t, float64(50), response["limit"])
	assert.Equal(t, float64(0), response["offset"])

	mockService.AssertExpectations(t)
}

func TestAuditHandler_GetAuditEntry(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	entry := &audit.AuditEntry{
		ID:        "entry123",
		EventType: audit.EventUserLogin,
		Level:     audit.AuditLevelInfo,
		UserID:    stringPtr("user123"),
		Resource:  "/auth/login",
	}

	mockService.On("GetAuditEntry", mock.Anything, "entry123").Return(entry, nil)

	req, _ := http.NewRequest("GET", "/api/audit/logs/entry123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response audit.AuditEntry
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "entry123", response.ID)
	assert.Equal(t, audit.EventUserLogin, response.EventType)

	mockService.AssertExpectations(t)
}

func TestAuditHandler_GetUserActivity(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	entries := []audit.AuditEntry{
		{
			ID:        "entry1",
			EventType: audit.EventUserLogin,
			UserID:    stringPtr("user123"),
		},
		{
			ID:        "entry2",
			EventType: audit.EventDocumentUploaded,
			UserID:    stringPtr("user123"),
		},
	}

	mockService.On("GetUserActivity",
		mock.Anything,
		"user123",
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
	).Return(entries, nil)

	req, _ := http.NewRequest("GET", "/api/audit/users/user123/activity", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "user123", response["user_id"])
	assert.Equal(t, float64(2), response["count"])

	mockService.AssertExpectations(t)
}

func TestAuditHandler_GetDataLineage(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	lineage := []audit.DataLineageEntry{
		{
			ID:             "lineage1",
			DataID:         "doc123",
			DataType:       "document",
			Transformation: "text_extraction",
		},
		{
			ID:             "lineage2",
			DataID:         "doc123",
			DataType:       "document",
			Transformation: "embedding_generation",
		},
	}

	mockService.On("GetDataLineage", mock.Anything, "doc123").Return(lineage, nil)

	req, _ := http.NewRequest("GET", "/api/audit/lineage/doc123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "doc123", response["data_id"])

	lineageResponse := response["lineage"].([]interface{})
	assert.Len(t, lineageResponse, 2)

	mockService.AssertExpectations(t)
}

func TestAuditHandler_GenerateAuditReport(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	report := &audit.AuditReport{
		ID:          "report123",
		Title:       "Test Report",
		GeneratedAt: time.Now(),
		Format:      "json",
		Summary: audit.ReportSummary{
			TotalEvents: 100,
		},
	}

	mockService.On("GenerateAuditReport",
		mock.Anything,
		mock.MatchedBy(func(q audit.AuditQuery) bool {
			return q.StartDate != nil && q.EndDate != nil
		}),
		"json",
	).Return(report, nil)

	requestBody := map[string]interface{}{
		"start_date": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"end_date":   time.Now().Format(time.RFC3339),
		"format":     "json",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/audit/reports", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response audit.AuditReport
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "report123", response.ID)
	assert.Equal(t, "json", response.Format)

	mockService.AssertExpectations(t)
}

func TestAuditHandler_ExportAuditReport(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	exportData := []byte(`{"report": "data"}`)

	mockService.On("ExportAuditReport", mock.Anything, "report123", "json").Return(exportData, nil)

	req, _ := http.NewRequest("GET", "/api/audit/reports/report123/export?format=json", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "audit_report_report123.json")
	assert.Equal(t, exportData, w.Body.Bytes())

	mockService.AssertExpectations(t)
}

func TestAuditHandler_GetSecurityAlerts(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	alerts := []audit.SecurityAlert{
		{
			ID:        "alert1",
			AlertType: "EXCESSIVE_FAILED_LOGINS",
			Severity:  audit.AuditLevelWarning,
			Status:    "open",
		},
		{
			ID:        "alert2",
			AlertType: "MULTIPLE_IP_LOGINS",
			Severity:  audit.AuditLevelWarning,
			Status:    "investigating",
		},
	}

	mockService.On("GetSecurityAlerts", mock.Anything, "open", 20, 0).Return(alerts, nil)

	req, _ := http.NewRequest("GET", "/api/audit/security/alerts?status=open", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	alertsResponse := response["alerts"].([]interface{})
	assert.Len(t, alertsResponse, 2)

	mockService.AssertExpectations(t)
}

func TestAuditHandler_UpdateSecurityAlert(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	resolution := "False positive - legitimate user behavior"
	mockService.On("UpdateSecurityAlert", mock.Anything, "alert123", "resolved", &resolution).Return(nil)

	requestBody := map[string]interface{}{
		"status":     "resolved",
		"resolution": resolution,
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("PUT", "/api/audit/security/alerts/alert123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "alert123", response["alert_id"])
	assert.Equal(t, "resolved", response["status"])

	mockService.AssertExpectations(t)
}

func TestAuditHandler_DetectAnomalies(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	alerts := []audit.SecurityAlert{
		{
			ID:        "anomaly1",
			AlertType: "EXCESSIVE_FAILED_LOGINS",
			Severity:  audit.AuditLevelWarning,
		},
	}

	mockService.On("DetectAnomalies",
		mock.Anything,
		"user123",
		10*time.Minute,
	).Return(alerts, nil)

	requestBody := map[string]interface{}{
		"user_id":     "user123",
		"time_window": "10m",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/audit/security/anomalies", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "user123", response["user_id"])
	assert.Equal(t, float64(1), response["count"])

	mockService.AssertExpectations(t)
}

func TestAuditHandler_GenerateComplianceReport(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	report := &audit.ComplianceReport{
		ID:              "compliance123",
		Standard:        "FISMA",
		ComplianceScore: 85.5,
		Status:          "compliant",
	}

	mockService.On("GenerateComplianceReport",
		mock.Anything,
		"FISMA",
		mock.MatchedBy(func(p audit.ReportPeriod) bool {
			return !p.StartDate.IsZero() && !p.EndDate.IsZero()
		}),
	).Return(report, nil)

	requestBody := map[string]interface{}{
		"standard":   "FISMA",
		"start_date": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		"end_date":   time.Now().Format(time.RFC3339),
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/audit/compliance/reports", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response audit.ComplianceReport
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "compliance123", response.ID)
	assert.Equal(t, "FISMA", response.Standard)
	assert.Equal(t, 85.5, response.ComplianceScore)

	mockService.AssertExpectations(t)
}

func TestAuditHandler_ValidateCompliance(t *testing.T) {
	mockService := new(MockAuditService)
	router := setupAuditRouter(mockService)

	report := &audit.ComplianceReport{
		ID:              "validation123",
		Standard:        "NIST",
		ComplianceScore: 92.0,
		Status:          "compliant",
	}

	mockService.On("ValidateCompliance", mock.Anything, "NIST").Return(report, nil)

	req, _ := http.NewRequest("GET", "/api/audit/compliance/validate/NIST", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response audit.ComplianceReport
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation123", response.ID)
	assert.Equal(t, "NIST", response.Standard)
	assert.Equal(t, 92.0, response.ComplianceScore)

	mockService.AssertExpectations(t)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
