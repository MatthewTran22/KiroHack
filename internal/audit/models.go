package audit

import (
	"time"
)

// AuditLevel represents the severity/importance of an audit event
type AuditLevel string

const (
	AuditLevelInfo     AuditLevel = "INFO"
	AuditLevelWarning  AuditLevel = "WARNING"
	AuditLevelError    AuditLevel = "ERROR"
	AuditLevelCritical AuditLevel = "CRITICAL"
	AuditLevelSecurity AuditLevel = "SECURITY"
)

// AuditEventType categorizes different types of audit events
type AuditEventType string

const (
	// User actions
	EventUserLogin             AuditEventType = "USER_LOGIN"
	EventUserLogout            AuditEventType = "USER_LOGOUT"
	EventUserLoginFailed       AuditEventType = "USER_LOGIN_FAILED"
	EventUserCreated           AuditEventType = "USER_CREATED"
	EventUserUpdated           AuditEventType = "USER_UPDATED"
	EventUserDeleted           AuditEventType = "USER_DELETED"
	EventUserPermissionChanged AuditEventType = "USER_PERMISSION_CHANGED"

	// Document operations
	EventDocumentUploaded  AuditEventType = "DOCUMENT_UPLOADED"
	EventDocumentProcessed AuditEventType = "DOCUMENT_PROCESSED"
	EventDocumentAccessed  AuditEventType = "DOCUMENT_ACCESSED"
	EventDocumentDeleted   AuditEventType = "DOCUMENT_DELETED"
	EventDocumentUpdated   AuditEventType = "DOCUMENT_UPDATED"

	// Consultation operations
	EventConsultationStarted     AuditEventType = "CONSULTATION_STARTED"
	EventConsultationCompleted   AuditEventType = "CONSULTATION_COMPLETED"
	EventRecommendationGenerated AuditEventType = "RECOMMENDATION_GENERATED"
	EventRecommendationAccessed  AuditEventType = "RECOMMENDATION_ACCESSED"

	// Knowledge base operations
	EventKnowledgeAdded    AuditEventType = "KNOWLEDGE_ADDED"
	EventKnowledgeUpdated  AuditEventType = "KNOWLEDGE_UPDATED"
	EventKnowledgeDeleted  AuditEventType = "KNOWLEDGE_DELETED"
	EventKnowledgeAccessed AuditEventType = "KNOWLEDGE_ACCESSED"

	// System operations
	EventSystemStartup        AuditEventType = "SYSTEM_STARTUP"
	EventSystemShutdown       AuditEventType = "SYSTEM_SHUTDOWN"
	EventConfigurationChanged AuditEventType = "CONFIGURATION_CHANGED"
	EventBackupCreated        AuditEventType = "BACKUP_CREATED"
	EventDataPurged           AuditEventType = "DATA_PURGED"

	// Security events
	EventSecurityViolation  AuditEventType = "SECURITY_VIOLATION"
	EventUnauthorizedAccess AuditEventType = "UNAUTHORIZED_ACCESS"
	EventSuspiciousActivity AuditEventType = "SUSPICIOUS_ACTIVITY"
	EventDataBreach         AuditEventType = "DATA_BREACH"
	EventEncryptionFailure  AuditEventType = "ENCRYPTION_FAILURE"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID           string                 `json:"id" bson:"_id"`
	Timestamp    time.Time              `json:"timestamp" bson:"timestamp"`
	EventType    AuditEventType         `json:"event_type" bson:"event_type"`
	Level        AuditLevel             `json:"level" bson:"level"`
	UserID       *string                `json:"user_id,omitempty" bson:"user_id,omitempty"`
	SessionID    *string                `json:"session_id,omitempty" bson:"session_id,omitempty"`
	IPAddress    string                 `json:"ip_address" bson:"ip_address"`
	UserAgent    string                 `json:"user_agent" bson:"user_agent"`
	Resource     string                 `json:"resource" bson:"resource"`
	Action       string                 `json:"action" bson:"action"`
	Result       string                 `json:"result" bson:"result"` // "success", "failure", "partial"
	Details      map[string]interface{} `json:"details" bson:"details"`
	RequestID    string                 `json:"request_id" bson:"request_id"`
	Duration     *time.Duration         `json:"duration,omitempty" bson:"duration,omitempty"`
	ErrorCode    *string                `json:"error_code,omitempty" bson:"error_code,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty" bson:"error_message,omitempty"`
}

// DataLineageEntry tracks the flow and transformation of data
type DataLineageEntry struct {
	ID             string                 `json:"id" bson:"_id"`
	DataID         string                 `json:"data_id" bson:"data_id"`
	DataType       string                 `json:"data_type" bson:"data_type"` // "document", "knowledge", "recommendation"
	SourceID       *string                `json:"source_id,omitempty" bson:"source_id,omitempty"`
	SourceType     *string                `json:"source_type,omitempty" bson:"source_type,omitempty"`
	Transformation string                 `json:"transformation" bson:"transformation"`
	ProcessedBy    string                 `json:"processed_by" bson:"processed_by"` // service/component name
	ProcessedAt    time.Time              `json:"processed_at" bson:"processed_at"`
	Version        int                    `json:"version" bson:"version"`
	Metadata       map[string]interface{} `json:"metadata" bson:"metadata"`
	QualityMetrics map[string]float64     `json:"quality_metrics" bson:"quality_metrics"`
}

// AuditReport represents a generated audit report
type AuditReport struct {
	ID          string                 `json:"id" bson:"_id"`
	Title       string                 `json:"title" bson:"title"`
	Description string                 `json:"description" bson:"description"`
	GeneratedBy string                 `json:"generated_by" bson:"generated_by"`
	GeneratedAt time.Time              `json:"generated_at" bson:"generated_at"`
	StartDate   time.Time              `json:"start_date" bson:"start_date"`
	EndDate     time.Time              `json:"end_date" bson:"end_date"`
	Filters     map[string]interface{} `json:"filters" bson:"filters"`
	Summary     ReportSummary          `json:"summary" bson:"summary"`
	Entries     []AuditEntry           `json:"entries" bson:"entries"`
	Format      string                 `json:"format" bson:"format"` // "json", "csv", "pdf"
	FilePath    *string                `json:"file_path,omitempty" bson:"file_path,omitempty"`
}

// ReportSummary provides statistical summary of audit events
type ReportSummary struct {
	TotalEvents         int                    `json:"total_events" bson:"total_events"`
	EventsByType        map[AuditEventType]int `json:"events_by_type" bson:"events_by_type"`
	EventsByLevel       map[AuditLevel]int     `json:"events_by_level" bson:"events_by_level"`
	EventsByResult      map[string]int         `json:"events_by_result" bson:"events_by_result"`
	UniqueUsers         int                    `json:"unique_users" bson:"unique_users"`
	UniqueResources     int                    `json:"unique_resources" bson:"unique_resources"`
	SecurityEvents      int                    `json:"security_events" bson:"security_events"`
	FailedOperations    int                    `json:"failed_operations" bson:"failed_operations"`
	AverageResponseTime *time.Duration         `json:"average_response_time,omitempty" bson:"average_response_time,omitempty"`
}

// AuditQuery represents search criteria for audit logs
type AuditQuery struct {
	StartDate   *time.Time       `json:"start_date,omitempty"`
	EndDate     *time.Time       `json:"end_date,omitempty"`
	EventTypes  []AuditEventType `json:"event_types,omitempty"`
	Levels      []AuditLevel     `json:"levels,omitempty"`
	UserIDs     []string         `json:"user_ids,omitempty"`
	Resources   []string         `json:"resources,omitempty"`
	Results     []string         `json:"results,omitempty"`
	IPAddresses []string         `json:"ip_addresses,omitempty"`
	SearchText  *string          `json:"search_text,omitempty"`
	Limit       int              `json:"limit"`
	Offset      int              `json:"offset"`
	SortBy      string           `json:"sort_by"`
	SortOrder   string           `json:"sort_order"` // "asc", "desc"
}

// SecurityAlert represents a security event that requires attention
type SecurityAlert struct {
	ID          string                 `json:"id" bson:"_id"`
	Timestamp   time.Time              `json:"timestamp" bson:"timestamp"`
	AlertType   string                 `json:"alert_type" bson:"alert_type"`
	Severity    AuditLevel             `json:"severity" bson:"severity"`
	Title       string                 `json:"title" bson:"title"`
	Description string                 `json:"description" bson:"description"`
	UserID      *string                `json:"user_id,omitempty" bson:"user_id,omitempty"`
	IPAddress   string                 `json:"ip_address" bson:"ip_address"`
	Resource    string                 `json:"resource" bson:"resource"`
	Details     map[string]interface{} `json:"details" bson:"details"`
	Status      string                 `json:"status" bson:"status"` // "open", "investigating", "resolved", "false_positive"
	AssignedTo  *string                `json:"assigned_to,omitempty" bson:"assigned_to,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty" bson:"resolved_at,omitempty"`
	Resolution  *string                `json:"resolution,omitempty" bson:"resolution,omitempty"`
}

// ComplianceReport represents a compliance audit report
type ComplianceReport struct {
	ID              string                  `json:"id" bson:"_id"`
	Standard        string                  `json:"standard" bson:"standard"` // "FISMA", "FedRAMP", "NIST", etc.
	GeneratedBy     string                  `json:"generated_by" bson:"generated_by"`
	GeneratedAt     time.Time               `json:"generated_at" bson:"generated_at"`
	ReportPeriod    ReportPeriod            `json:"report_period" bson:"report_period"`
	ComplianceScore float64                 `json:"compliance_score" bson:"compliance_score"`
	Requirements    []ComplianceRequirement `json:"requirements" bson:"requirements"`
	Findings        []ComplianceFinding     `json:"findings" bson:"findings"`
	Recommendations []string                `json:"recommendations" bson:"recommendations"`
	Status          string                  `json:"status" bson:"status"` // "compliant", "non_compliant", "partial"
}

// ReportPeriod defines the time range for a compliance report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date" bson:"start_date"`
	EndDate   time.Time `json:"end_date" bson:"end_date"`
}

// ComplianceRequirement represents a specific compliance requirement
type ComplianceRequirement struct {
	ID          string   `json:"id" bson:"id"`
	Title       string   `json:"title" bson:"title"`
	Description string   `json:"description" bson:"description"`
	Status      string   `json:"status" bson:"status"` // "met", "not_met", "partial", "not_applicable"
	Score       float64  `json:"score" bson:"score"`
	Evidence    []string `json:"evidence" bson:"evidence"`
}

// ComplianceFinding represents a compliance issue or observation
type ComplianceFinding struct {
	ID          string     `json:"id" bson:"id"`
	Severity    AuditLevel `json:"severity" bson:"severity"`
	Title       string     `json:"title" bson:"title"`
	Description string     `json:"description" bson:"description"`
	Requirement string     `json:"requirement" bson:"requirement"`
	Evidence    []string   `json:"evidence" bson:"evidence"`
	Remediation string     `json:"remediation" bson:"remediation"`
	DueDate     *time.Time `json:"due_date,omitempty" bson:"due_date,omitempty"`
}
