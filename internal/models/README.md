# Data Models

This package contains the core data models for the AI Government Consultant application, along with MongoDB integration and comprehensive validation.

## Overview

The data models are designed to support a government consulting platform that processes documents, manages knowledge, handles user consultations, and maintains audit trails. All models include proper BSON tags for MongoDB serialization and comprehensive validation methods.

## Core Models

### Document Model (`document.go`)

Represents documents uploaded to the system for analysis and consultation.

**Key Features:**
- Security classification support (PUBLIC, INTERNAL, CONFIDENTIAL, SECRET, TOP_SECRET)
- Document metadata with flexible custom fields
- Processing status tracking
- Vector embeddings for semantic search
- Entity extraction results
- Comprehensive validation

**Example Usage:**
```go
document := models.Document{
    Name:        "policy-document.pdf",
    ContentType: "application/pdf",
    Size:        1024000,
    UploadedBy:  userID,
    Classification: models.SecurityClassification{
        Level: "CONFIDENTIAL",
        Compartments: []string{"NOFORN"},
    },
    ProcessingStatus: models.ProcessingStatusCompleted,
}

if err := document.Validate(); err != nil {
    // Handle validation error
}
```

### User Model (`user.go`)

Represents users in the system with role-based access control and security clearances.

**Key Features:**
- Role-based permissions (Admin, Analyst, Manager, Viewer, Consultant)
- Security clearance levels matching government standards
- Multi-factor authentication support
- Fine-grained permission system
- Classification-based access control

**Example Usage:**
```go
user := models.User{
    Email:             "analyst@government.gov",
    Name:              "Jane Analyst",
    Role:              models.UserRoleAnalyst,
    SecurityClearance: models.SecurityClearanceConfidential,
    Permissions: []models.Permission{
        {Resource: "documents", Actions: []string{"read", "write"}},
    },
}

canAccess := user.CanAccessClassification("CONFIDENTIAL") // true
hasPermission := user.HasPermission("documents", "read")   // true
```

### Consultation Session Model (`consultation.go`)

Represents AI consultation sessions with comprehensive response tracking.

**Key Features:**
- Multiple consultation types (Policy, Strategy, Operations, Technology)
- Detailed recommendations with impact assessments
- Risk analysis and mitigation strategies
- Source attribution and confidence scoring
- Session context and history tracking

**Example Usage:**
```go
session := models.ConsultationSession{
    UserID: userID,
    Type:   models.ConsultationTypePolicy,
    Query:  "What are the FISMA compliance requirements?",
    Status: models.SessionStatusActive,
}

if session.IsCompleted() {
    // Process completed consultation
}
```

### Knowledge Item Model (`knowledge.go`)

Represents knowledge base entries with relationship management and validation.

**Key Features:**
- Multiple knowledge types (Facts, Rules, Procedures, Best Practices, etc.)
- Knowledge relationships and graph construction
- Validation and expiration tracking
- Usage analytics and effectiveness scoring
- Source attribution and reliability tracking

**Example Usage:**
```go
knowledge := models.KnowledgeItem{
    Content:    "FISMA requires comprehensive security controls",
    Type:       models.KnowledgeTypeRegulation,
    Title:      "FISMA Requirements",
    Confidence: 0.95,
    CreatedBy:  userID,
}

knowledge.IncrementUsage("policy_analysis")
knowledge.AddRelationship(models.RelationshipTypeRelatedTo, relatedID, 0.8, "context")
```

## Database Integration

### MongoDB Connection (`database/mongodb.go`)

Provides robust MongoDB connection management with:
- Connection pooling and timeout handling
- Health checks and monitoring
- Automatic index creation
- Error handling and recovery

**Configuration:**
```go
config := &database.Config{
    URI:            "mongodb://localhost:27017",
    DatabaseName:   "ai_government_consultant",
    ConnectTimeout: 10 * time.Second,
    MaxPoolSize:    100,
    MinPoolSize:    5,
}

mongodb, err := database.NewMongoDB(config)
```

### Database Initialization (`database/init.go`)

Handles database setup and seeding:
- Index creation for optimal query performance
- Default admin user creation
- Sample knowledge base entries
- Database migration support

## Indexes and Performance

The system creates comprehensive indexes for optimal query performance:

### Document Indexes
- Name, uploaded_by, upload_date
- Processing status and classification level
- Metadata categories and tags
- Full-text search on content
- Vector search indexes for embeddings

### User Indexes
- Unique email index
- Department, role, and security clearance
- Activity and login tracking

### Consultation Indexes
- User and type-based queries
- Status and date-based filtering
- Full-text search on queries

### Knowledge Indexes
- Type, category, and tag-based queries
- Confidence and validation status
- Full-text search on content
- Vector search for semantic similarity

## Validation and Error Handling

All models include comprehensive validation:
- Required field validation
- Data type and format validation
- Business rule validation
- Cross-field validation

**Error Types:**
- `ErrDocumentNameRequired`
- `ErrUserEmailRequired`
- `ErrConsultationQueryRequired`
- `ErrKnowledgeContentRequired`
- And many more...

## Testing

Comprehensive test coverage includes:
- Unit tests for all model validation
- BSON serialization/deserialization tests
- Business logic testing
- Database integration tests
- Performance and load testing

**Run Tests:**
```bash
go test ./internal/models -v
go test ./internal/database -v
```

## Security Features

### Data Classification
- Government-standard security classifications
- Compartment and handling restrictions
- Automatic access control enforcement

### User Security
- Role-based access control (RBAC)
- Security clearance validation
- Multi-factor authentication support
- Session management and tracking

### Audit Trail
- Comprehensive activity logging
- Data lineage tracking
- Change history maintenance
- Compliance reporting support

## Usage Examples

See `examples/models_demo.go` for comprehensive usage examples demonstrating:
- Model creation and validation
- Database connection and operations
- Security and permission checking
- Knowledge management operations

**Run Demo:**
```bash
go run examples/models_demo.go
```

## Dependencies

- `go.mongodb.org/mongo-driver` - MongoDB driver
- `go.mongodb.org/mongo-driver/bson` - BSON serialization
- `go.mongodb.org/mongo-driver/mongo/options` - MongoDB options

## Best Practices

1. **Always validate models** before database operations
2. **Use proper security classifications** for sensitive data
3. **Implement proper error handling** for all database operations
4. **Monitor performance** with appropriate indexes
5. **Follow government security standards** for data handling
6. **Maintain audit trails** for all operations
7. **Use connection pooling** for database efficiency
8. **Implement proper backup strategies** for data protection

## Future Enhancements

- Advanced vector search capabilities
- Real-time collaboration features
- Enhanced analytics and reporting
- Integration with external government systems
- Advanced AI model integration
- Automated compliance checking