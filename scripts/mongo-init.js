// MongoDB initialization script for development
db = db.getSiblingDB('ai_government_consultant');

// Create collections with proper indexes
db.createCollection('users');
db.createCollection('documents');
db.createCollection('knowledge_items');
db.createCollection('consultations');
db.createCollection('audit_logs');
db.createCollection('system_config');

// Create indexes for users collection
db.users.createIndex({ "email": 1 }, { unique: true });
db.users.createIndex({ "department": 1 });
db.users.createIndex({ "role": 1 });
db.users.createIndex({ "security_clearance.level": 1 });

// Create indexes for documents collection
db.documents.createIndex({ "uploaded_by": 1 });
db.documents.createIndex({ "uploaded_at": -1 });
db.documents.createIndex({ "classification.level": 1 });
db.documents.createIndex({ "metadata.category": 1 });
db.documents.createIndex({ "metadata.tags": 1 });
db.documents.createIndex({ "processing_status": 1 });

// Create vector search index for documents (requires MongoDB Atlas or Enterprise)
// This will be created programmatically when vector search is implemented
// db.documents.createSearchIndex({
//   "name": "document_vector_index",
//   "definition": {
//     "fields": [
//       {
//         "type": "vector",
//         "path": "embeddings",
//         "numDimensions": 1536,
//         "similarity": "cosine"
//       }
//     ]
//   }
// });

// Create indexes for knowledge_items collection
db.knowledge_items.createIndex({ "type": 1 });
db.knowledge_items.createIndex({ "source.document_id": 1 });
db.knowledge_items.createIndex({ "tags": 1 });
db.knowledge_items.createIndex({ "last_updated": -1 });
db.knowledge_items.createIndex({ "confidence": -1 });

// Create vector search index for knowledge items
// db.knowledge_items.createSearchIndex({
//   "name": "knowledge_vector_index",
//   "definition": {
//     "fields": [
//       {
//         "type": "vector",
//         "path": "embeddings",
//         "numDimensions": 1536,
//         "similarity": "cosine"
//       }
//     ]
//   }
// });

// Create indexes for consultations collection
db.consultations.createIndex({ "user_id": 1 });
db.consultations.createIndex({ "type": 1 });
db.consultations.createIndex({ "created_at": -1 });
db.consultations.createIndex({ "status": 1 });

// Create indexes for audit_logs collection
db.audit_logs.createIndex({ "user_id": 1 });
db.audit_logs.createIndex({ "action": 1 });
db.audit_logs.createIndex({ "timestamp": -1 });
db.audit_logs.createIndex({ "resource": 1 });
db.audit_logs.createIndex({ "result": 1 });

// Create compound indexes for common queries
db.documents.createIndex({ "uploaded_by": 1, "uploaded_at": -1 });
db.consultations.createIndex({ "user_id": 1, "created_at": -1 });
db.audit_logs.createIndex({ "user_id": 1, "timestamp": -1 });

// Insert initial system configuration
db.system_config.insertOne({
  "_id": "app_config",
  "version": "1.0.0",
  "features": {
    "document_processing": true,
    "ai_consultation": true,
    "knowledge_management": true,
    "audit_logging": true
  },
  "limits": {
    "max_document_size": 50 * 1024 * 1024, // 50MB
    "max_documents_per_user": 1000,
    "consultation_rate_limit": 100, // per hour
    "max_consultation_history": 10000
  },
  "security": {
    "password_min_length": 12,
    "session_timeout": 3600, // 1 hour
    "max_login_attempts": 5,
    "lockout_duration": 900 // 15 minutes
  },
  "created_at": new Date(),
  "updated_at": new Date()
});

print('Database initialization completed successfully');
print('Collections created: users, documents, knowledge_items, consultations, audit_logs, system_config');
print('Indexes created for optimal query performance');
print('Initial system configuration inserted');