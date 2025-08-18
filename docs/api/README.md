# AI Government Consultant API

This document provides an overview of the REST API for the AI Government Consultant platform.

## Overview

The AI Government Consultant API provides endpoints for:

- **Authentication & Authorization**: User management, login, logout, and access control
- **Document Management**: Upload, process, search, and manage government documents
- **AI Consultations**: Create consultation sessions and get AI-powered recommendations
- **Knowledge Management**: Manage the knowledge base with facts, procedures, and policies
- **Audit & Reporting**: Track user activities and generate compliance reports

## Base URL

- Development: `http://localhost:8080/api/v1`
- Production: `https://api.ai-gov-consultant.com/v1`

## Authentication

The API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### Getting a Token

1. Register a new user or use existing credentials
2. Login with email and password
3. Use the returned access token for subsequent requests

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

## Quick Start

### 1. Health Check

```bash
curl http://localhost:8080/health
```

### 2. Register a User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "analyst@gov.com",
    "name": "Government Analyst",
    "department": "Policy Analysis",
    "role": "analyst",
    "security_clearance": "secret",
    "password": "securepassword123"
  }'
```

### 3. Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "analyst@gov.com",
    "password": "securepassword123"
  }'
```

### 4. Upload a Document

```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Authorization: Bearer <your-token>" \
  -F "file=@policy-document.pdf" \
  -F "category=policy" \
  -F "title=New Policy Document" \
  -F "tags=policy,government,analysis"
```

### 5. Create a Consultation

```bash
curl -X POST http://localhost:8080/api/v1/consultations \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What are the key considerations for implementing a new data privacy policy?",
    "type": "policy",
    "max_sources": 10
  }'
```

### 6. Create Knowledge Item

```bash
curl -X POST http://localhost:8080/api/v1/knowledge \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Data Privacy Best Practice",
    "content": "Always encrypt sensitive data both in transit and at rest",
    "type": "best_practice",
    "category": "security",
    "tags": ["privacy", "security", "encryption"]
  }'
```

## API Endpoints

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout
- `GET /auth/profile` - Get user profile
- `PUT /auth/profile` - Update user profile

### Documents
- `GET /documents` - List documents
- `POST /documents` - Upload document
- `GET /documents/{id}` - Get document
- `PUT /documents/{id}` - Update document
- `DELETE /documents/{id}` - Delete document
- `POST /documents/search` - Search documents

### Consultations
- `GET /consultations` - List consultations
- `POST /consultations` - Create consultation
- `GET /consultations/{id}` - Get consultation
- `POST /consultations/{id}/continue` - Continue multi-turn consultation
- `GET /consultations/history` - Get consultation history

### Knowledge Management
- `GET /knowledge` - List knowledge items
- `POST /knowledge` - Create knowledge item
- `GET /knowledge/{id}` - Get knowledge item
- `PUT /knowledge/{id}` - Update knowledge item
- `DELETE /knowledge/{id}` - Delete knowledge item
- `POST /knowledge/search` - Search knowledge
- `GET /knowledge/categories` - Get categories
- `GET /knowledge/types` - Get knowledge types

### Audit & Reporting
- `GET /audit/logs` - Get audit logs
- `GET /audit/logs/{id}` - Get specific audit log
- `POST /audit/reports/generate` - Generate audit report
- `GET /audit/activity/users/{id}` - Get user activity
- `GET /audit/activity/system` - Get system activity

## Response Format

### Success Response
```json
{
  "message": "Operation completed successfully",
  "data": {
    // Response data
  }
}
```

### Error Response
```json
{
  "error": "Error type",
  "message": "Detailed error message",
  "code": "ERROR_CODE"
}
```

## Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Rate Limiting

The API implements rate limiting to ensure fair usage:

- **Default**: 60 requests per minute per user
- **Burst**: Up to 10 requests in a short burst
- **Headers**: Rate limit information is included in response headers

## Security

### Authentication
- JWT tokens with configurable expiration
- Refresh token mechanism for long-lived sessions
- Multi-factor authentication support

### Authorization
- Role-based access control (RBAC)
- Security clearance levels
- Resource-level permissions

### Data Protection
- All data encrypted in transit (TLS 1.3)
- Sensitive data encrypted at rest
- Audit logging for all operations

## Error Handling

The API provides detailed error messages and appropriate HTTP status codes. Common error scenarios:

### Authentication Errors
```json
{
  "error": "Invalid credentials",
  "code": "INVALID_CREDENTIALS"
}
```

### Validation Errors
```json
{
  "error": "Invalid request",
  "message": "Field 'email' is required",
  "code": "INVALID_REQUEST"
}
```

### Permission Errors
```json
{
  "error": "Insufficient permissions",
  "code": "INSUFFICIENT_PERMISSIONS"
}
```

## Testing

### Integration Tests

Run the integration tests with Docker containers:

```bash
# Start test dependencies
docker-compose -f docker-compose.test.yml up -d

# Run tests
go test ./test/integration/... -v

# Clean up
docker-compose -f docker-compose.test.yml down
```

### Manual Testing

Use the provided Postman collection or curl commands to test endpoints manually.

## OpenAPI Specification

The complete API specification is available in OpenAPI 3.0 format:
- [OpenAPI YAML](./openapi.yaml)
- Interactive documentation available at `/api/v1/docs`

## Support

For API support and questions:
- Email: support@ai-gov-consultant.com
- Documentation: [API Docs](./openapi.yaml)
- Issues: GitHub Issues