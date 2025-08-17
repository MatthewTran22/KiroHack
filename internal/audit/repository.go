package audit

import (
	"context"
	"time"
)

// Repository defines the interface for audit data persistence
type Repository interface {
	// Audit entries
	CreateAuditEntry(ctx context.Context, entry AuditEntry) error
	GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error)
	SearchAuditEntries(ctx context.Context, query AuditQuery) ([]AuditEntry, int, error)
	DeleteAuditEntriesBefore(ctx context.Context, beforeDate time.Time) (int, error)
	ArchiveAuditEntries(ctx context.Context, beforeDate time.Time) error

	// Data lineage
	CreateDataLineageEntry(ctx context.Context, entry DataLineageEntry) error
	GetDataLineage(ctx context.Context, dataID string) ([]DataLineageEntry, error)
	GetDataProvenance(ctx context.Context, dataID string) (*DataLineageEntry, error)

	// Audit reports
	CreateAuditReport(ctx context.Context, report AuditReport) error
	GetAuditReport(ctx context.Context, reportID string) (*AuditReport, error)
	ListAuditReports(ctx context.Context, limit, offset int) ([]AuditReport, error)

	// Security alerts
	CreateSecurityAlert(ctx context.Context, alert SecurityAlert) error
	GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]SecurityAlert, error)
	UpdateSecurityAlert(ctx context.Context, alertID string, updates map[string]interface{}) error

	// Compliance reports
	CreateComplianceReport(ctx context.Context, report ComplianceReport) error
	GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error)
}
