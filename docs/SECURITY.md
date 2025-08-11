# Security Configuration Guide

This document outlines the security configurations for the AI Government Consultant platform, with specific focus on database and service security.

## Database Security

### MongoDB Security Configuration

#### Current Setup Analysis

**Development Configuration:**
- **Port Binding**: `127.0.0.1:27017:27017` (localhost only)
- **Authentication**: Enabled with root user
- **Network**: Isolated within Docker network
- **External Access**: Limited to localhost for development tools

**Production Configuration:**
- **Port Binding**: No external ports exposed
- **Authentication**: Strong passwords via environment variables
- **Network**: Internal Docker network only
- **Logging**: Enhanced logging with rotation
- **Performance**: Optimized cache and storage settings

#### Security Features Implemented

1. **Authentication Required**: `--auth` flag enabled
2. **Bind IP Configuration**: Controlled via `--bind_ip` parameter
3. **Logging**: Comprehensive audit logging enabled
4. **Resource Limits**: Memory and CPU limits in production
5. **Network Isolation**: Services communicate via internal Docker network

### Redis Security Configuration

#### Security Features

1. **Password Protection**: `--requirepass` enabled
2. **Protected Mode**: Enabled for additional security
3. **Bind Configuration**: Controlled network binding
4. **Memory Limits**: Prevents resource exhaustion
5. **Persistence**: AOF and RDB snapshots configured

## Network Security

### Docker Network Configuration

```yaml
networks:
  app-network:
    driver: bridge
```

**Security Benefits:**
- **Isolation**: Services isolated from host network
- **Internal Communication**: Services communicate via service names
- **No External Access**: Database services not exposed in production

### Port Exposure Strategy

#### Development Environment
```yaml
# Allow external access for development tools
mongodb:
  ports:
    - "27017:27017"  # Accessible from host
redis:
  ports:
    - "6379:6379"    # Accessible from host
```

#### Production Environment
```yaml
# No external ports - internal communication only
mongodb:
  ports: []  # No external access
redis:
  ports: []  # No external access
```

## Environment-Specific Configurations

### Development Setup

```bash
# Start with development overrides
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Enable development tools
docker-compose --profile dev up -d
```

**Development Features:**
- External database access for tools
- Development admin interfaces (mongo-express, redis-commander)
- Debug logging enabled
- Weaker passwords for convenience

### Production Setup

```bash
# Start with production overrides
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

**Production Features:**
- No external database access
- Strong environment-based passwords
- Resource limits and scaling
- Enhanced logging and monitoring
- Development tools disabled

## Security Checklist

### Pre-Production Security Review

- [ ] **Database Access**
  - [ ] External ports removed from production
  - [ ] Strong passwords configured via environment variables
  - [ ] Authentication enabled on all services
  - [ ] Network isolation verified

- [ ] **Application Security**
  - [ ] JWT secrets changed from defaults
  - [ ] HTTPS enabled with valid certificates
  - [ ] Security headers configured
  - [ ] CORS policies properly configured

- [ ] **Infrastructure Security**
  - [ ] Firewall rules configured
  - [ ] VPN access for administrative tasks
  - [ ] Regular security updates scheduled
  - [ ] Backup encryption enabled

- [ ] **Monitoring and Logging**
  - [ ] Audit logging enabled
  - [ ] Log rotation configured
  - [ ] Security monitoring alerts set up
  - [ ] Failed authentication tracking

## Environment Variables for Production

Create a `.env.prod` file with secure values:

```bash
# JWT Configuration
JWT_ACCESS_SECRET=your-very-secure-access-secret-256-bits-minimum
JWT_REFRESH_SECRET=your-very-secure-refresh-secret-256-bits-minimum

# Database Configuration
MONGO_ROOT_PASSWORD=your-very-secure-mongo-root-password
MONGO_APP_PASSWORD=your-secure-app-user-password

# Redis Configuration
REDIS_PASSWORD=your-very-secure-redis-password

# Application Configuration
LOG_LEVEL=warn
ENVIRONMENT=production
```

## Database User Management

### MongoDB User Setup

For production, create dedicated application users with minimal privileges:

```javascript
// Connect as admin
use ai_government_consultant

// Create application user with limited privileges
db.createUser({
  user: "app_user",
  pwd: "secure_app_password",
  roles: [
    { role: "readWrite", db: "ai_government_consultant" }
  ]
})

// Create read-only user for monitoring
db.createUser({
  user: "monitor_user", 
  pwd: "secure_monitor_password",
  roles: [
    { role: "read", db: "ai_government_consultant" }
  ]
})
```

### Redis Security

Redis security is handled through:
- Password authentication (`requirepass`)
- Protected mode (prevents external access without auth)
- Network binding restrictions
- Memory limits to prevent DoS

## Monitoring and Alerting

### Security Monitoring

1. **Failed Authentication Attempts**
   - Monitor MongoDB authentication failures
   - Track Redis unauthorized access attempts
   - Alert on JWT token validation failures

2. **Resource Usage**
   - Monitor database connection counts
   - Track memory and CPU usage
   - Alert on unusual resource consumption

3. **Network Security**
   - Monitor external connection attempts
   - Track unusual network patterns
   - Alert on port scanning activities

### Log Analysis

Key security events to monitor:
- Authentication failures
- Privilege escalation attempts
- Unusual query patterns
- Resource exhaustion events
- Network intrusion attempts

## Backup Security

### Database Backups

```bash
# Encrypted MongoDB backup
mongodump --uri="mongodb://backup_user:password@localhost:27017/ai_government_consultant" \
  --gzip --archive=backup.gz

# Encrypt backup file
gpg --cipher-algo AES256 --compress-algo 1 --s2k-mode 3 \
  --s2k-digest-algo SHA512 --s2k-count 65536 --symmetric backup.gz
```

### Redis Backups

```bash
# Redis backup with encryption
redis-cli --rdb dump.rdb
gpg --cipher-algo AES256 --symmetric dump.rdb
```

## Compliance Considerations

### Government Security Standards

1. **FISMA Compliance**
   - Implement continuous monitoring
   - Regular security assessments
   - Incident response procedures

2. **NIST Cybersecurity Framework**
   - Identify: Asset inventory and risk assessment
   - Protect: Access controls and data protection
   - Detect: Security monitoring and alerting
   - Respond: Incident response procedures
   - Recover: Backup and recovery procedures

3. **FedRAMP Requirements**
   - Continuous monitoring
   - Regular vulnerability assessments
   - Strong authentication requirements
   - Data encryption at rest and in transit

## Regular Security Tasks

### Daily
- Review authentication logs
- Monitor resource usage
- Check backup completion

### Weekly  
- Review security alerts
- Update security patches
- Rotate temporary credentials

### Monthly
- Security assessment review
- Update security documentation
- Review user access permissions

### Quarterly
- Penetration testing
- Security training updates
- Disaster recovery testing

This security configuration provides defense-in-depth protection suitable for government applications while maintaining operational efficiency.