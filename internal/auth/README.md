# Authentication and Authorization System

This package provides a comprehensive authentication and authorization system for the AI Government Consultant platform. It includes JWT token management, password hashing, multi-factor authentication (MFA), role-based access control (RBAC), and session management with Redis.

## Features

- **JWT Token Management**: Access and refresh tokens with configurable expiration
- **Password Security**: Bcrypt hashing with configurable cost and password strength validation
- **Multi-Factor Authentication**: TOTP-based MFA with QR code generation and backup codes
- **Role-Based Access Control**: Fine-grained permissions system with security clearance levels
- **Session Management**: Redis-based session storage with token blacklisting
- **Gin Middleware**: Ready-to-use middleware for HTTP routes
- **Comprehensive Testing**: Full test coverage for all components

## Quick Start

### 1. Setup Dependencies

```go
import (
    "ai-government-consultant/internal/auth"
    "github.com/redis/go-redis/v9"
    "go.mongodb.org/mongo-driver/mongo"
)

// MongoDB collection for users
userCollection := db.Collection("users")

// Redis client for sessions
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// JWT configuration
jwtConfig := auth.JWTConfig{
    AccessSecret:  "your-access-secret",
    RefreshSecret: "your-refresh-secret",
    AccessTTL:     15 * time.Minute,
    RefreshTTL:    24 * time.Hour,
    SessionTTL:    30 * time.Minute,
    BlacklistTTL:  24 * time.Hour,
    Issuer:        "your-app-name",
}
```

### 2. Create Services

```go
// Create authentication service
authService := auth.NewAuthService(userCollection, redisClient, jwtConfig)

// Create authorization service
authzService := auth.NewAuthorizationService()

// Create middleware
middleware := auth.NewAuthMiddleware(authService, authzService)
```

### 3. Setup Routes

```go
router := gin.New()

// Add security middleware
router.Use(middleware.CORS())
router.Use(middleware.SecurityHeaders())
router.Use(middleware.AuditLog())

// Public routes
public := router.Group("/api/v1/public")
{
    public.POST("/register", registerHandler)
    public.POST("/login", loginHandler)
    public.POST("/refresh", refreshTokenHandler)
}

// Protected routes
protected := router.Group("/api/v1")
protected.Use(middleware.RequireAuth())
{
    protected.GET("/profile", profileHandler)
    protected.POST("/logout", logoutHandler)
}

// Admin routes
admin := router.Group("/api/v1/admin")
admin.Use(middleware.RequireAuth())
admin.Use(middleware.RequireRole(models.UserRoleAdmin))
{
    admin.GET("/users", listUsersHandler)
}

// Permission-based routes
documents := router.Group("/api/v1/documents")
documents.Use(middleware.RequireAuth())
{
    documents.GET("/", 
        middleware.RequirePermission(auth.ResourceDocuments, auth.ActionRead),
        listDocumentsHandler)
    
    documents.POST("/", 
        middleware.RequirePermission(auth.ResourceDocuments, auth.ActionWrite),
        createDocumentHandler)
    
    // Security clearance required
    documents.GET("/classified/:id",
        middleware.RequirePermissionWithClearance(
            auth.ResourceDocuments, 
            auth.ActionRead, 
            "SECRET"),
        getClassifiedDocumentHandler)
}
```

## Core Components

### AuthService

The main authentication service that handles user registration, login, token management, and MFA.

```go
// Register a new user
user := &models.User{
    Email:             "user@example.com",
    Name:              "John Doe",
    Department:        "IT",
    Role:              models.UserRoleAnalyst,
    SecurityClearance: models.SecurityClearanceSecret,
}
err := authService.RegisterUser(ctx, user, "password123")

// Authenticate user
credentials := auth.AuthCredentials{
    Email:    "user@example.com",
    Password: "password123",
    MFACode:  "123456", // Optional, required if MFA is enabled
}
result, err := authService.Authenticate(ctx, credentials, "127.0.0.1", "user-agent")

// Validate token
validation, err := authService.ValidateToken(ctx, accessToken)

// Refresh token
newTokens, err := authService.RefreshToken(ctx, refreshToken)

// Logout
err := authService.Logout(ctx, accessToken, sessionID)
```

### AuthorizationService

Handles role-based access control and permission checking.

```go
// Check if user has permission
authorized, err := authzService.Authorize(ctx, user, auth.ResourceDocuments, auth.ActionRead)

// Check with security clearance
authorized, err := authzService.AuthorizeWithSecurityClearance(
    ctx, user, auth.ResourceDocuments, auth.ActionRead, "SECRET")

// Get user permissions
permissions, err := authzService.GetUserPermissions(ctx, user)

// Check document access
canAccess, err := authzService.CanUserAccessDocument(ctx, user, "SECRET", auth.ActionRead)
```

### Middleware

Gin middleware for protecting routes and checking permissions.

```go
// Require authentication
router.Use(middleware.RequireAuth())

// Require specific role
router.Use(middleware.RequireRole(models.UserRoleAdmin))

// Require permission
router.Use(middleware.RequirePermission(auth.ResourceDocuments, auth.ActionRead))

// Require security clearance
router.Use(middleware.RequireSecurityClearance("SECRET"))

// Combined permission and clearance
router.Use(middleware.RequirePermissionWithClearance(
    auth.ResourceDocuments, auth.ActionRead, "SECRET"))

// Optional authentication (doesn't fail if no token)
router.Use(middleware.OptionalAuth())

// Rate limiting per user
router.Use(middleware.RateLimitByUser(100)) // 100 requests per minute
```

## User Roles and Permissions

### Default Roles

- **Admin**: Full access to all resources
- **Manager**: Read/write access to most resources, limited admin functions
- **Analyst**: Read/write access to documents, consultations, and knowledge
- **Consultant**: Read/write access to consultations and knowledge, read-only documents
- **Viewer**: Read-only access to documents, consultations, and knowledge

### Security Clearance Levels

- **PUBLIC**: Accessible to all users
- **INTERNAL**: Accessible to internal users and above
- **CONFIDENTIAL**: Accessible to confidential clearance and above
- **SECRET**: Accessible to secret clearance and above
- **TOP_SECRET**: Accessible only to top secret clearance

### Resources

- **documents**: Document management
- **consultations**: AI consultation sessions
- **users**: User management
- **knowledge**: Knowledge base management
- **audit**: Audit logs and reports
- **system**: System administration

### Actions

- **read**: View/retrieve resources
- **write**: Create/update resources
- **delete**: Remove resources
- **admin**: Administrative operations

## Multi-Factor Authentication

### Setup MFA

```go
// Generate MFA secret and QR code
mfaSetup, err := authService.SetupMFA(ctx, userID)
// Returns: secret, QR code URL, backup codes

// Enable MFA after user scans QR code
err := authService.EnableMFA(ctx, userID, "123456") // TOTP code

// Disable MFA
err := authService.DisableMFA(ctx, userID, "password")
```

### MFA Service

```go
mfaService := auth.NewMFAService("your-app-name")

// Generate secret
secret, err := mfaService.GenerateSecret()

// Generate QR code URL
qrURL := mfaService.GenerateQRCodeURL(secret, "user@example.com")

// Generate and validate codes
code, err := mfaService.GenerateCode(secret)
valid, err := mfaService.ValidateCode(secret, "123456")

// Generate backup codes
backupCodes, err := mfaService.GenerateBackupCodes(10)
```

## Session Management

Sessions are stored in Redis and provide additional security features:

```go
sessionService := auth.NewSessionService(redisClient, sessionTTL, blacklistTTL)

// Create session
sessionData := &auth.SessionData{
    UserID:       userID,
    Email:        "user@example.com",
    Role:         "analyst",
    LoginTime:    time.Now(),
    IPAddress:    "127.0.0.1",
    UserAgent:    "browser",
    MFAVerified:  true,
}
err := sessionService.CreateSession(ctx, sessionID, sessionData)

// Get session
session, err := sessionService.GetSession(ctx, sessionID)

// Update last activity
err := sessionService.UpdateLastActivity(ctx, sessionID)

// Blacklist token
err := sessionService.BlacklistToken(ctx, tokenID)

// Check if token is blacklisted
blacklisted, err := sessionService.IsTokenBlacklisted(ctx, tokenID)

// Invalidate all user sessions
err := sessionService.InvalidateUserSessions(ctx, userID)
```

## Password Security

The password service provides secure password hashing and validation:

```go
passwordService := auth.NewPasswordService()

// Hash password
hash, err := passwordService.HashPassword("password123")

// Verify password
err := passwordService.VerifyPassword("password123", hash)

// Validate password strength
err := passwordService.ValidatePassword("password123")

// Generate secure password
password, err := passwordService.GenerateSecurePassword(16)

// Generate salt
salt, err := passwordService.GenerateSalt(32)

// Secure comparison
equal := passwordService.SecureCompare("string1", "string2")
```

## Error Handling

The package defines comprehensive error types:

```go
// Authentication errors
models.ErrInvalidCredentials
models.ErrUserInactive
models.ErrInvalidToken
models.ErrTokenExpired
models.ErrTokenBlacklisted
models.ErrMFARequired
models.ErrInvalidMFACode

// Authorization errors
models.ErrInsufficientPermissions
models.ErrAccessDenied
models.ErrInvalidSecurityClearance

// User errors
models.ErrUserNotFound
models.ErrUserAlreadyExists
```

## Testing

The package includes comprehensive tests for all components:

```bash
# Run all auth tests
go test ./internal/auth -v

# Run specific test suites
go test ./internal/auth -v -run TestJWTService
go test ./internal/auth -v -run TestPasswordService
go test ./internal/auth -v -run TestMFAService
go test ./internal/auth -v -run TestAuthorizationService

# Run benchmarks
go test ./internal/auth -bench=.
```

## Configuration

### Environment Variables

```bash
# JWT Configuration
JWT_ACCESS_SECRET=your-access-secret-key
JWT_REFRESH_SECRET=your-refresh-secret-key
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=24h
JWT_SESSION_TTL=30m
JWT_BLACKLIST_TTL=24h
JWT_ISSUER=ai-government-consultant

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Password Configuration
PASSWORD_BCRYPT_COST=12
```

### Production Considerations

1. **Secrets Management**: Use a secure secret management system for JWT secrets
2. **Redis Security**: Enable Redis AUTH and use TLS in production
3. **Rate Limiting**: Implement proper rate limiting for authentication endpoints
4. **Monitoring**: Monitor authentication failures and suspicious activities
5. **Backup**: Regularly backup user data and session information
6. **Compliance**: Ensure compliance with government security standards

## Security Features

- **Password Strength**: Enforced password complexity requirements
- **Token Security**: Short-lived access tokens with secure refresh mechanism
- **Session Security**: Redis-based session management with automatic cleanup
- **MFA Support**: TOTP-based multi-factor authentication
- **Audit Logging**: Comprehensive audit trails for all authentication events
- **Rate Limiting**: Protection against brute force attacks
- **Security Headers**: Automatic security headers for HTTP responses
- **CORS Protection**: Configurable CORS policies
- **Token Blacklisting**: Immediate token invalidation on logout
- **Security Clearance**: Government-grade security classification system

This authentication system provides enterprise-grade security suitable for government applications while maintaining ease of use and comprehensive testing coverage.