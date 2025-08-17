# Audit System

The audit system provides comprehensive logging, monitoring, and compliance capabilities for the AI Government Consultant platform. It ensures all user actions, system operations, and security events are properly tracked and auditable according to government standards.

## Features

### Core Audit Logging
- **Comprehensive Event Tracking**: Logs all user actions, system operations, and security events
- **Structured Logging**: Uses standardized audit entry format with rich metadata
- **Automatic Context Capture**: Captures request IDs, IP addresses, user agents, and session information
- **Performance Tracking**: Records operation durations and response times

### Data Lineage and Provenance
- **Data Flow Tracking**: Tracks the complete lifecycle of data from ingestion to processing
- **Transformation History**: Records all data transformations and processing steps
- **Source Attribution**: Maintains links between derived data and original sources
- **Version Control**: Tracks data versions and changes over time

### Security Monitoring
- **Real-time Security Event Detection**: Monitors for unauthorized access, failed logins, and suspicious activities
- **Anomaly Detection**: Identifies unusual user behavior patterns
- **Security Alerts**: Generates and manages security alerts with severity levels
- **Threat Intelligence**: Correlates events to identify potential security threats

### Audit Reporting
- **Comprehensive Reports**: Generates detailed audit reports with statistical summaries
- **Multiple Export Formats**: Supports JSON and CSV export formats
- **Customizable Queries**: Flexible search and filtering capabilities
- **Scheduled Reports**: Automated report generation for compliance requirements

### Compliance Management
- **Government Standards**: Built-in support for FISMA, NIST, and other government compliance frameworks
- **Compliance Scoring**: Automated compliance assessment with scoring
- **Gap Analysis**: Identifies compliance gaps and provides remediation recommendations
- **Evidence Collection**: Automatically collects evidence for compliance audits

### Data Retention and Archival
- **Configurable Retention**: Flexible data retention policies based on government requirements
- **Automated Cleanup**: Automatic purging of old audit logs based on retention policies
- **Archival Support**: Long-term archival of audit data for historical compliance

## Architecture

### Components

1. **Service Layer** (`service.go`)
   - Core business logic for audit operations
   - Event processing and analysis
   - Report generation and compliance validation

2. **Repository Layer** (`repository.go`, `mongodb_repository.go`)
   - Data persistence abstraction
   - MongoDB implementation with optimized indexes
   - Efficient querying and aggregation

3. **Models** (`models.go`)
   - Structured data models for audit entries, reports, and alerts
   - Government compliance data structures
   - Security event classifications

4. **Middleware** (`middleware.go`)
   - Automatic audit logging for HTTP requests
   - Security event detection and alerting
   - Context enrichment with request metadata

5. **API Handlers** (`audit_handler.go`)
   - RESTful API endpoints for audit operations
   - Search, reporting, and compliance endpoints
   - Export and download functionality

6. **Export Utilities** (`export.go`)
   - Multi-format export capabilities
   - CSV and JSON report generation
   - Compliance report formatting

## Usage

### Basic Setup

```go
// Initialize MongoDB connection
db := mongodb.Connect("mongodb://localhost:27017/audit")

// Create repository and service
repo := audit.NewMongoRepository(db)
service := audit.NewService(repo)

// Set up middleware
router := gin.New()
router.Use(audit.AuditMiddleware(service))
router.Use(audit.SecurityEventMiddleware(service))
```

### Logging Events

```go
// Log user action
err := service.LogUserAction(ctx, "user123", "document_upload", "/documents", map[string]interface{}{
    "document_id": "doc456",
    "size": 1024,
})

// Log security event
err := service.LogSecurityEvent(ctx, audit.EventUnauthorizedAccess, &userID, "192.168.1.1", "/admin", details)

// Log system event
err := service.LogSystemEvent(ctx, audit.EventSystemStartup, map[string]interface{}{
    "version": "1.0.0",
})
```

### Data Lineage Tracking

```go
// Track data processing
lineage := audit.DataLineageEntry{
    DataID:         "doc123",
    DataType:       "document",
    SourceID:       &uploadID,
    Transformation: "text_extraction",
    ProcessedBy:    "document_service",
    Version:        1,
}

err := service.TrackDataLineage(ctx, lineage)

// Get complete lineage
lineage, err := service.GetDataLineage(ctx, "doc123")
```

### Searching Audit Logs

```go
query := audit.AuditQuery{
    StartDate:  &startDate,
    EndDate:    &endDate,
    UserIDs:    []string{"user123"},
    EventTypes: []audit.AuditEventType{audit.EventUserLogin, audit.EventDocumentUploaded},
    Limit:      100,
}

entries, total, err := service.SearchAuditLogs(ctx, query)
```

### Generating Reports

```go
// Generate audit report
report, err := service.GenerateAuditReport(ctx, query, "json")

// Export report
data, err := service.ExportAuditReport(ctx, report.ID, "csv")

// Generate compliance report
period := audit.ReportPeriod{
    StartDate: time.Now().AddDate(0, -1, 0),
    EndDate:   time.Now(),
}

complianceReport, err := service.GenerateComplianceReport(ctx, "FISMA", period)
```

### Security Monitoring

```go
// Detect anomalies
alerts, err := service.DetectAnomalies(ctx, "user123", 5*time.Minute)

// Get security alerts
alerts, err := service.GetSecurityAlerts(ctx, "open", 10, 0)

// Update alert status
err := service.UpdateSecurityAlert(ctx, alertID, "resolved", &resolution)
```

## API Endpoints

### Audit Logs
- `GET /api/audit/logs` - Search audit logs
- `GET /api/audit/logs/:id` - Get specific audit entry
- `GET /api/audit/users/:user_id/activity` - Get user activity

### Data Lineage
- `GET /api/audit/lineage/:data_id` - Get data lineage
- `GET /api/audit/provenance/:data_id` - Get data provenance

### Reports
- `POST /api/audit/reports` - Generate audit report
- `GET /api/audit/reports` - List audit reports
- `GET /api/audit/reports/:id` - Get audit report
- `GET /api/audit/reports/:id/export` - Export audit report

### Security
- `GET /api/audit/security/alerts` - Get security alerts
- `PUT /api/audit/security/alerts/:id` - Update security alert
- `POST /api/audit/security/anomalies` - Detect anomalies

### Compliance
- `POST /api/audit/compliance/reports` - Generate compliance report
- `GET /api/audit/compliance/reports/:id` - Get compliance report
- `GET /api/audit/compliance/validate/:standard` - Validate compliance

## Configuration

### Environment Variables

```bash
# MongoDB Configuration
MONGODB_URI=mongodb://localhost:27017
AUDIT_DATABASE_NAME=audit_db

# Retention Policies
AUDIT_RETENTION_DAYS=2555  # 7 years for government compliance
AUDIT_ARCHIVE_DAYS=365     # Archive after 1 year

# Security Settings
SECURITY_ALERT_THRESHOLD=5  # Failed login threshold
ANOMALY_DETECTION_WINDOW=5m # Time window for anomaly detection

# Compliance Standards
COMPLIANCE_STANDARDS=FISMA,NIST,FedRAMP
```

### Database Indexes

The system automatically creates optimized MongoDB indexes:

```javascript
// Audit logs indexes
db.audit_logs.createIndex({ "timestamp": -1 })
db.audit_logs.createIndex({ "user_id": 1, "timestamp": -1 })
db.audit_logs.createIndex({ "event_type": 1, "timestamp": -1 })
db.audit_logs.createIndex({ "level": 1, "timestamp": -1 })
db.audit_logs.createIndex({ "resource": 1, "timestamp": -1 })
db.audit_logs.createIndex({ "ip_address": 1, "timestamp": -1 })

// Data lineage indexes
db.data_lineage.createIndex({ "data_id": 1, "processed_at": -1 })
db.data_lineage.createIndex({ "source_id": 1 })

// Security alerts indexes
db.security_alerts.createIndex({ "timestamp": -1 })
db.security_alerts.createIndex({ "status": 1, "timestamp": -1 })
db.security_alerts.createIndex({ "severity": 1, "timestamp": -1 })
```

## Testing

### Unit Tests
```bash
go test ./internal/audit/...
```

### Integration Tests
```bash
go test ./test/integration/audit_test.go
```

### API Tests
```bash
go test ./internal/api/audit_handler_test.go
```

## Compliance Standards

### FISMA (Federal Information Security Management Act)
- Comprehensive audit logging (AU-2, AU-3, AU-6)
- Access control monitoring (AC-2, AC-3)
- Incident response tracking (IR-4, IR-5)
- Configuration management (CM-3, CM-6)

### NIST Cybersecurity Framework
- Identity and Access Management (PR.AC)
- Data Security (PR.DS)
- Information Protection Processes (PR.IP)
- Anomalies and Events (DE.AE)

### FedRAMP
- Continuous monitoring requirements
- Security assessment and authorization
- Incident response and reporting
- Configuration management

## Performance Considerations

### Scalability
- Horizontal scaling through MongoDB sharding
- Asynchronous audit logging to prevent blocking
- Efficient indexing for fast queries
- Batch processing for large operations

### Storage Optimization
- Configurable data retention policies
- Automatic archival and compression
- Efficient document storage formats
- Index optimization for query patterns

### Monitoring
- Performance metrics collection
- Query performance monitoring
- Storage usage tracking
- Alert response time monitoring

## Security

### Data Protection
- Encryption at rest and in transit
- Access control based on user roles
- Audit log integrity protection
- Secure export and transmission

### Privacy
- PII detection and masking
- Data classification and handling
- Retention policy enforcement
- Secure data disposal

### Compliance
- Government-approved encryption standards
- Audit trail immutability
- Chain of custody maintenance
- Evidence preservation

## Troubleshooting

### Common Issues

1. **High Storage Usage**
   - Check retention policies
   - Review archival settings
   - Monitor index sizes
   - Consider data compression

2. **Slow Query Performance**
   - Verify index usage
   - Optimize query patterns
   - Check MongoDB performance
   - Review aggregation pipelines

3. **Missing Audit Entries**
   - Check middleware configuration
   - Verify service initialization
   - Review error logs
   - Test audit logging manually

4. **Compliance Failures**
   - Review compliance requirements
   - Check audit coverage
   - Verify retention policies
   - Update compliance rules

### Monitoring and Alerting

- Set up monitoring for audit service health
- Configure alerts for compliance violations
- Monitor storage usage and performance
- Track security event frequencies

## Future Enhancements

- Machine learning-based anomaly detection
- Advanced threat intelligence integration
- Real-time compliance monitoring
- Enhanced visualization and dashboards
- Integration with SIEM systems
- Blockchain-based audit trail integrity