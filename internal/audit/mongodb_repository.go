package audit

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongoRepository implements the Repository interface using MongoDB
type mongoRepository struct {
	db                   *mongo.Database
	auditCollection      *mongo.Collection
	lineageCollection    *mongo.Collection
	reportsCollection    *mongo.Collection
	alertsCollection     *mongo.Collection
	complianceCollection *mongo.Collection
}

// NewMongoRepository creates a new MongoDB-based audit repository
func NewMongoRepository(db *mongo.Database) Repository {
	repo := &mongoRepository{
		db:                   db,
		auditCollection:      db.Collection("audit_logs"),
		lineageCollection:    db.Collection("data_lineage"),
		reportsCollection:    db.Collection("audit_reports"),
		alertsCollection:     db.Collection("security_alerts"),
		complianceCollection: db.Collection("compliance_reports"),
	}

	// Create indexes for better performance
	repo.createIndexes()

	return repo
}

// createIndexes creates necessary database indexes
func (r *mongoRepository) createIndexes() {
	ctx := context.Background()

	// Audit logs indexes
	auditIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "event_type", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "level", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "resource", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "ip_address", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "request_id", Value: 1},
			},
		},
	}

	// Data lineage indexes
	lineageIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "data_id", Value: 1},
				{Key: "processed_at", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "source_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "data_type", Value: 1},
				{Key: "processed_at", Value: -1},
			},
		},
	}

	// Security alerts indexes
	alertIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "severity", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
	}

	// Create indexes (ignore errors if indexes already exist)
	r.auditCollection.Indexes().CreateMany(ctx, auditIndexes)
	r.lineageCollection.Indexes().CreateMany(ctx, lineageIndexes)
	r.alertsCollection.Indexes().CreateMany(ctx, alertIndexes)
}

// CreateAuditEntry creates a new audit entry
func (r *mongoRepository) CreateAuditEntry(ctx context.Context, entry AuditEntry) error {
	_, err := r.auditCollection.InsertOne(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}
	return nil
}

// GetAuditEntry retrieves a specific audit entry by ID
func (r *mongoRepository) GetAuditEntry(ctx context.Context, id string) (*AuditEntry, error) {
	var entry AuditEntry
	err := r.auditCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("audit entry not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get audit entry: %w", err)
	}
	return &entry, nil
}

// SearchAuditEntries searches audit entries based on query criteria
func (r *mongoRepository) SearchAuditEntries(ctx context.Context, query AuditQuery) ([]AuditEntry, int, error) {
	filter := r.buildAuditFilter(query)

	// Count total matching documents
	total, err := r.auditCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit entries: %w", err)
	}

	// Build find options
	findOptions := options.Find()

	if query.Limit > 0 {
		findOptions.SetLimit(int64(query.Limit))
	}

	if query.Offset > 0 {
		findOptions.SetSkip(int64(query.Offset))
	}

	// Set sort order
	sortOrder := 1
	if query.SortOrder == "desc" {
		sortOrder = -1
	}

	sortField := "timestamp"
	if query.SortBy != "" {
		sortField = query.SortBy
	}

	findOptions.SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	// Execute query
	cursor, err := r.auditCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search audit entries: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []AuditEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, 0, fmt.Errorf("failed to decode audit entries: %w", err)
	}

	return entries, int(total), nil
}

// buildAuditFilter builds MongoDB filter from AuditQuery
func (r *mongoRepository) buildAuditFilter(query AuditQuery) bson.M {
	filter := bson.M{}

	// Date range filter
	if query.StartDate != nil || query.EndDate != nil {
		dateFilter := bson.M{}
		if query.StartDate != nil {
			dateFilter["$gte"] = *query.StartDate
		}
		if query.EndDate != nil {
			dateFilter["$lte"] = *query.EndDate
		}
		filter["timestamp"] = dateFilter
	}

	// Event types filter
	if len(query.EventTypes) > 0 {
		filter["event_type"] = bson.M{"$in": query.EventTypes}
	}

	// Levels filter
	if len(query.Levels) > 0 {
		filter["level"] = bson.M{"$in": query.Levels}
	}

	// User IDs filter
	if len(query.UserIDs) > 0 {
		filter["user_id"] = bson.M{"$in": query.UserIDs}
	}

	// Resources filter
	if len(query.Resources) > 0 {
		filter["resource"] = bson.M{"$in": query.Resources}
	}

	// Results filter
	if len(query.Results) > 0 {
		filter["result"] = bson.M{"$in": query.Results}
	}

	// IP addresses filter
	if len(query.IPAddresses) > 0 {
		filter["ip_address"] = bson.M{"$in": query.IPAddresses}
	}

	// Text search filter
	if query.SearchText != nil && *query.SearchText != "" {
		filter["$or"] = []bson.M{
			{"action": primitive.Regex{Pattern: *query.SearchText, Options: "i"}},
			{"resource": primitive.Regex{Pattern: *query.SearchText, Options: "i"}},
			{"details": primitive.Regex{Pattern: *query.SearchText, Options: "i"}},
		}
	}

	return filter
}

// DeleteAuditEntriesBefore deletes audit entries before a specific date
func (r *mongoRepository) DeleteAuditEntriesBefore(ctx context.Context, beforeDate time.Time) (int, error) {
	filter := bson.M{"timestamp": bson.M{"$lt": beforeDate}}

	result, err := r.auditCollection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit entries: %w", err)
	}

	return int(result.DeletedCount), nil
}

// ArchiveAuditEntries archives audit entries before a specific date
func (r *mongoRepository) ArchiveAuditEntries(ctx context.Context, beforeDate time.Time) error {
	// In a real implementation, this would move entries to an archive collection
	// For now, we'll just mark them as archived
	filter := bson.M{"timestamp": bson.M{"$lt": beforeDate}}
	update := bson.M{"$set": bson.M{"archived": true, "archived_at": time.Now()}}

	_, err := r.auditCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to archive audit entries: %w", err)
	}

	return nil
}

// CreateDataLineageEntry creates a new data lineage entry
func (r *mongoRepository) CreateDataLineageEntry(ctx context.Context, entry DataLineageEntry) error {
	_, err := r.lineageCollection.InsertOne(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to create data lineage entry: %w", err)
	}
	return nil
}

// GetDataLineage retrieves the complete lineage for a data item
func (r *mongoRepository) GetDataLineage(ctx context.Context, dataID string) ([]DataLineageEntry, error) {
	filter := bson.M{"data_id": dataID}
	findOptions := options.Find().SetSort(bson.D{{Key: "processed_at", Value: -1}})

	cursor, err := r.lineageCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get data lineage: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []DataLineageEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode data lineage entries: %w", err)
	}

	return entries, nil
}

// GetDataProvenance retrieves the immediate provenance (source) of a data item
func (r *mongoRepository) GetDataProvenance(ctx context.Context, dataID string) (*DataLineageEntry, error) {
	filter := bson.M{"data_id": dataID}
	findOptions := options.FindOne().SetSort(bson.D{{Key: "processed_at", Value: -1}})

	var entry DataLineageEntry
	err := r.lineageCollection.FindOne(ctx, filter, findOptions).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("data provenance not found for: %s", dataID)
		}
		return nil, fmt.Errorf("failed to get data provenance: %w", err)
	}

	return &entry, nil
}

// CreateAuditReport creates a new audit report
func (r *mongoRepository) CreateAuditReport(ctx context.Context, report AuditReport) error {
	_, err := r.reportsCollection.InsertOne(ctx, report)
	if err != nil {
		return fmt.Errorf("failed to create audit report: %w", err)
	}
	return nil
}

// GetAuditReport retrieves a specific audit report
func (r *mongoRepository) GetAuditReport(ctx context.Context, reportID string) (*AuditReport, error) {
	var report AuditReport
	err := r.reportsCollection.FindOne(ctx, bson.M{"_id": reportID}).Decode(&report)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("audit report not found: %s", reportID)
		}
		return nil, fmt.Errorf("failed to get audit report: %w", err)
	}
	return &report, nil
}

// ListAuditReports lists all audit reports with pagination
func (r *mongoRepository) ListAuditReports(ctx context.Context, limit, offset int) ([]AuditReport, error) {
	findOptions := options.Find().
		SetSort(bson.D{{Key: "generated_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.reportsCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit reports: %w", err)
	}
	defer cursor.Close(ctx)

	var reports []AuditReport
	if err := cursor.All(ctx, &reports); err != nil {
		return nil, fmt.Errorf("failed to decode audit reports: %w", err)
	}

	return reports, nil
}

// CreateSecurityAlert creates a new security alert
func (r *mongoRepository) CreateSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	_, err := r.alertsCollection.InsertOne(ctx, alert)
	if err != nil {
		return fmt.Errorf("failed to create security alert: %w", err)
	}
	return nil
}

// GetSecurityAlerts retrieves security alerts with optional status filter
func (r *mongoRepository) GetSecurityAlerts(ctx context.Context, status string, limit, offset int) ([]SecurityAlert, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.alertsCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get security alerts: %w", err)
	}
	defer cursor.Close(ctx)

	var alerts []SecurityAlert
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, fmt.Errorf("failed to decode security alerts: %w", err)
	}

	return alerts, nil
}

// UpdateSecurityAlert updates a security alert
func (r *mongoRepository) UpdateSecurityAlert(ctx context.Context, alertID string, updates map[string]interface{}) error {
	filter := bson.M{"_id": alertID}
	update := bson.M{"$set": updates}

	result, err := r.alertsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update security alert: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("security alert not found: %s", alertID)
	}

	return nil
}

// CreateComplianceReport creates a new compliance report
func (r *mongoRepository) CreateComplianceReport(ctx context.Context, report ComplianceReport) error {
	_, err := r.complianceCollection.InsertOne(ctx, report)
	if err != nil {
		return fmt.Errorf("failed to create compliance report: %w", err)
	}
	return nil
}

// GetComplianceReport retrieves a specific compliance report
func (r *mongoRepository) GetComplianceReport(ctx context.Context, reportID string) (*ComplianceReport, error) {
	var report ComplianceReport
	err := r.complianceCollection.FindOne(ctx, bson.M{"_id": reportID}).Decode(&report)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("compliance report not found: %s", reportID)
		}
		return nil, fmt.Errorf("failed to get compliance report: %w", err)
	}
	return &report, nil
}
