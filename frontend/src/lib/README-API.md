# API Client Implementation

This document describes the comprehensive API client implementation for the AI Government Consultant frontend.

## Overview

The API client provides a type-safe, robust interface for communicating with the backend services. It includes comprehensive error handling, retry logic, request/response interceptors, and user-friendly error messages.

## Features

### Core Features

- **Type-safe API calls** with TypeScript interfaces
- **Automatic authentication** with JWT token management
- **Request/response interceptors** for cross-cutting concerns
- **Comprehensive error handling** with user-friendly messages
- **Retry logic** with exponential backoff
- **Timeout management** with configurable timeouts
- **Token refresh** with automatic retry of failed requests

### API Endpoints

The client provides organized endpoints for:

- **Authentication** (`auth`) - Login, logout, token refresh, MFA
- **Documents** (`documents`) - Upload, retrieve, search, delete
- **Consultations** (`consultations`) - Create sessions, send messages, export
- **Users** (`users`) - User management (admin only)
- **Audit** (`audit`) - Audit logs and compliance reporting

## Usage

### Basic Usage

```typescript
import { apiClient } from '@/lib/api';

// Authentication
const authResponse = await apiClient.auth.login({
  email: 'user@example.com',
  password: 'password123',
});

// Documents
const documents = await apiClient.documents.getDocuments({
  category: 'policy',
  tags: ['important'],
});

// Consultations
const consultation = await apiClient.consultations.createSession({
  type: 'policy',
  title: 'Policy Analysis',
  priority: 'high',
});
```

### Error Handling

```typescript
import { APIError, getErrorMessage } from '@/lib/api';

try {
  await apiClient.documents.upload(files);
} catch (error) {
  if (error instanceof APIError) {
    const userError = getErrorMessage(error);
    console.log(userError.title, userError.message);
    
    if (userError.action === 'retry') {
      // Show retry button
    }
  }
}
```

### Custom Configuration

```typescript
// Custom timeout and retry configuration
const result = await apiClient.request('/api/custom', {
  method: 'POST',
  timeout: 60000, // 60 seconds
  retry: {
    maxAttempts: 5,
    baseDelay: 2000,
  },
});
```

## API Client Architecture

### Class Structure

```typescript
class APIClient {
  // Core request method with interceptors and retry logic
  private async request<T>(endpoint: string, options: RequestConfig): Promise<T>
  
  // Organized API endpoints
  auth: AuthAPI
  documents: DocumentsAPI
  consultations: ConsultationsAPI
  users: UsersAPI
  audit: AuditAPI
  
  // Interceptor management
  addRequestInterceptor(interceptor: RequestInterceptor): void
  addResponseInterceptor(interceptor: ResponseInterceptor): void
}
```

### Request Flow

1. **Request Preparation**
   - Apply request interceptors
   - Add authentication headers
   - Set default headers and timeout

2. **Request Execution**
   - Execute HTTP request with timeout
   - Handle network errors and timeouts

3. **Response Processing**
   - Apply response interceptors
   - Handle HTTP errors
   - Parse JSON response

4. **Error Handling**
   - Create APIError instances
   - Apply error interceptors
   - Handle token refresh for 401 errors

5. **Retry Logic**
   - Determine if error is retryable
   - Apply exponential backoff
   - Retry up to configured maximum attempts

## Error Handling

### APIError Class

```typescript
class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public details?: Record<string, unknown>,
    public requestId?: string
  )
  
  // Convenience properties
  get isNetworkError(): boolean
  get isAuthError(): boolean
  get isServerError(): boolean
  get isClientError(): boolean
  get isRetryable(): boolean
}
```

### Error Categories

- **Network Errors** (status: 0) - Connection failures, timeouts
- **Authentication Errors** (status: 401, 403) - Invalid credentials, expired tokens
- **Validation Errors** (status: 400) - Invalid input, file validation
- **Not Found Errors** (status: 404) - Resource not found
- **Rate Limiting** (status: 429) - Too many requests
- **Server Errors** (status: 5xx) - Internal server errors

### User-Friendly Error Messages

The `getErrorMessage` function converts technical API errors into user-friendly messages:

```typescript
interface UserFriendlyError {
  title: string;
  message: string;
  action?: string;
  actionLabel?: string;
}
```

## Interceptors

### Request Interceptors

Automatically applied to all requests:

- **Authentication Interceptor** - Adds JWT tokens to requests
- **Custom Headers** - Adds application-specific headers

### Response Interceptors

Handle cross-cutting response concerns:

- **Token Refresh Interceptor** - Automatically refreshes expired tokens
- **Error Transformation** - Converts HTTP errors to APIError instances

## Retry Logic

### Configuration

```typescript
interface RetryConfig {
  maxAttempts: number;      // Maximum retry attempts (default: 3)
  baseDelay: number;        // Base delay in ms (default: 1000)
  maxDelay: number;         // Maximum delay in ms (default: 10000)
  backoffFactor: number;    // Exponential backoff factor (default: 2)
  retryableStatuses: number[]; // HTTP status codes to retry
}
```

### Retry Strategy

- **Exponential Backoff** - Delay increases exponentially with each attempt
- **Jitter** - Random variation to prevent thundering herd
- **Status-based** - Only retry specific HTTP status codes
- **Error-based** - Retry network errors and server errors

## Testing

### Unit Tests

Comprehensive unit tests cover:

- All API endpoints and methods
- Error handling scenarios
- Retry logic and timeouts
- Request/response interceptors
- User-friendly error messages

### Integration Tests

Integration tests with Docker containers verify:

- Full authentication flow
- Document upload and management
- Consultation creation and messaging
- Error handling with real backend
- Performance under load

### Running Tests

```bash
# Unit tests
npm test -- --testPathPatterns="api.test.ts"
npm test -- --testPathPatterns="error-messages.test.ts"

# Integration tests (requires Docker)
npm run test:api-integration
```

## Configuration

### Environment Variables

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080  # Backend API URL
NODE_ENV=development                       # Environment
```

### Default Configuration

```typescript
const defaultRetryConfig: RetryConfig = {
  maxAttempts: 3,
  baseDelay: 1000,
  maxDelay: 10000,
  backoffFactor: 2,
  retryableStatuses: [408, 429, 500, 502, 503, 504],
};
```

## Security Considerations

### Token Management

- JWT tokens stored securely using tokenManager
- Automatic token refresh before expiration
- Secure token transmission in Authorization headers
- Token cleanup on logout

### Request Security

- CSRF protection through custom headers
- Input validation and sanitization
- Secure file upload validation
- Request timeout to prevent hanging requests

### Error Information

- Sensitive information filtered from error messages
- Request IDs for debugging without exposing internals
- User-friendly messages that don't reveal system details

## Performance Optimizations

### Request Optimization

- Connection reuse through fetch API
- Request deduplication for identical requests
- Timeout management to prevent resource leaks
- Efficient retry logic with exponential backoff

### Memory Management

- Proper cleanup of AbortController instances
- Efficient error object creation
- Minimal memory footprint for interceptors

## Monitoring and Debugging

### Request Logging

- Comprehensive error logging with context
- Request/response timing information
- Retry attempt tracking
- Token refresh events

### Error Tracking

- Structured error information for monitoring
- Request ID correlation across services
- User-friendly error categorization
- Performance metrics collection

## Future Enhancements

### Planned Features

- Request caching with TTL
- Request deduplication
- Offline support with request queuing
- GraphQL support
- WebSocket integration for real-time features
- Request/response compression
- Advanced retry strategies (circuit breaker)

### Performance Improvements

- Request batching for bulk operations
- Connection pooling optimization
- Response streaming for large payloads
- Progressive loading for large datasets

This API client implementation provides a robust, type-safe, and user-friendly interface for all backend communication needs while maintaining high performance and reliability standards.