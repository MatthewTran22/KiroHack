# API Keys and External Services Requirements

This document outlines all the API keys, external services, and configuration requirements needed to make the AI Government Consultant platform fully operational.

## 🔑 Required API Keys

### 1. Large Language Model (LLM) Services

#### Google Gemini (Recommended)
- **Service**: Google AI Studio / Vertex AI
- **Environment Variable**: `LLM_API_KEY`
- **Purpose**: Primary AI consultation engine, document analysis, knowledge extraction
- **How to obtain**:
  1. Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
  2. Create a new API key
  3. Set usage limits and billing
- **Pricing**: Pay-per-use, free tier available
- **Required for**: Core AI functionality

#### Alternative LLM Providers (Choose One)
- **OpenAI GPT-4/GPT-3.5**
  - Environment Variable: `OPENAI_API_KEY`
  - Website: [OpenAI Platform](https://platform.openai.com/api-keys)
  
- **Anthropic Claude**
  - Environment Variable: `ANTHROPIC_API_KEY`
  - Website: [Anthropic Console](https://console.anthropic.com/)
  
- **Azure OpenAI**
  - Environment Variables: `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`
  - Website: [Azure Portal](https://portal.azure.com/)

### 2. Document Processing Services

#### PDF Processing (Optional Enhancement)
- **Service**: Adobe PDF Services API
- **Environment Variable**: `ADOBE_PDF_API_KEY`
- **Purpose**: Advanced PDF text extraction and processing
- **How to obtain**:
  1. Visit [Adobe Developer Console](https://developer.adobe.com/console)
  2. Create a new project
  3. Add PDF Services API
- **Alternative**: Use open-source libraries like `unidoc` or `pdfcpu`

#### Office Document Processing (Optional Enhancement)
- **Service**: Microsoft Graph API
- **Environment Variables**: `MICROSOFT_CLIENT_ID`, `MICROSOFT_CLIENT_SECRET`
- **Purpose**: Enhanced DOC/DOCX processing
- **How to obtain**:
  1. Visit [Azure App Registrations](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps)
  2. Register a new application
  3. Generate client secret

### 3. Email Services (For Notifications)

#### SendGrid (Recommended)
- **Environment Variable**: `SENDGRID_API_KEY`
- **Purpose**: Email notifications, alerts, reports
- **How to obtain**:
  1. Visit [SendGrid](https://sendgrid.com/)
  2. Create account and verify domain
  3. Generate API key in Settings > API Keys
- **Free tier**: 100 emails/day

#### Alternative Email Services
- **AWS SES**
  - Environment Variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`
  - Website: [AWS SES Console](https://console.aws.amazon.com/ses/)
  
- **Mailgun**
  - Environment Variable: `MAILGUN_API_KEY`
  - Website: [Mailgun](https://www.mailgun.com/)

### 4. Authentication Services (Optional Enhancement)

#### OAuth Providers
- **Google OAuth**
  - Environment Variables: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
  - Website: [Google Cloud Console](https://console.cloud.google.com/)
  
- **Microsoft Azure AD**
  - Environment Variables: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
  - Website: [Azure Portal](https://portal.azure.com/)
  
- **GitHub OAuth**
  - Environment Variables: `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`
  - Website: [GitHub Developer Settings](https://github.com/settings/developers)

### 5. Monitoring and Analytics

#### Application Performance Monitoring
- **New Relic**
  - Environment Variable: `NEW_RELIC_LICENSE_KEY`
  - Website: [New Relic](https://newrelic.com/)
  
- **DataDog**
  - Environment Variable: `DATADOG_API_KEY`
  - Website: [DataDog](https://www.datadoghq.com/)

#### Error Tracking
- **Sentry**
  - Environment Variable: `SENTRY_DSN`
  - Website: [Sentry](https://sentry.io/)
  - Purpose: Error tracking and performance monitoring

### 6. Search and Analytics (Optional Enhancement)

#### Elasticsearch Service
- **Elastic Cloud**
  - Environment Variables: `ELASTICSEARCH_URL`, `ELASTICSEARCH_API_KEY`
  - Website: [Elastic Cloud](https://cloud.elastic.co/)
  - Purpose: Advanced search capabilities

#### Vector Database (For Semantic Search)
- **Pinecone**
  - Environment Variable: `PINECONE_API_KEY`
  - Website: [Pinecone](https://www.pinecone.io/)
  - Purpose: Vector embeddings for semantic document search
  
- **Weaviate Cloud**
  - Environment Variable: `WEAVIATE_API_KEY`
  - Website: [Weaviate](https://weaviate.io/)

## 🗄️ Database Services

### Primary Database
- **MongoDB Atlas** (Recommended for production)
  - Environment Variable: `MONGO_URI`
  - Website: [MongoDB Atlas](https://www.mongodb.com/cloud/atlas)
  - Free tier: 512MB storage
  - Connection string format: `mongodb+srv://username:password@cluster.mongodb.net/database`

### Cache/Session Store
- **Redis Cloud** (Recommended for production)
  - Environment Variables: `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
  - Website: [Redis Cloud](https://redis.com/redis-enterprise-cloud/)
  - Free tier: 30MB

## 🔐 Security Services

### SSL/TLS Certificates
- **Let's Encrypt** (Free)
  - Automated certificate management
  - Use with reverse proxy (Nginx, Traefik)
  
- **Cloudflare** (Recommended)
  - Environment Variable: `CLOUDFLARE_API_TOKEN`
  - Website: [Cloudflare](https://www.cloudflare.com/)
  - Free tier includes SSL, DDoS protection, CDN

### Secrets Management
- **HashiCorp Vault**
  - Environment Variables: `VAULT_ADDR`, `VAULT_TOKEN`
  - Website: [Vault](https://www.vaultproject.io/)
  
- **AWS Secrets Manager**
  - Environment Variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
  - Website: [AWS Console](https://console.aws.amazon.com/secretsmanager/)

## 📊 Analytics and Logging

### Log Management
- **LogDNA/Mezmo**
  - Environment Variable: `LOGDNA_INGESTION_KEY`
  - Website: [Mezmo](https://www.mezmo.com/)
  
- **Papertrail**
  - Environment Variable: `PAPERTRAIL_API_TOKEN`
  - Website: [Papertrail](https://www.papertrail.com/)

## 🌐 CDN and Storage

### File Storage
- **AWS S3**
  - Environment Variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_S3_BUCKET`
  - Website: [AWS S3 Console](https://console.aws.amazon.com/s3/)
  - Purpose: Document storage, backups
  
- **Google Cloud Storage**
  - Environment Variable: `GOOGLE_CLOUD_STORAGE_BUCKET`
  - Website: [Google Cloud Console](https://console.cloud.google.com/)

### Content Delivery Network
- **Cloudflare CDN** (Free tier available)
- **AWS CloudFront**
- **Google Cloud CDN**

## 🔧 Development and CI/CD

### Version Control and CI/CD
- **GitHub Actions** (Free for public repos)
  - Environment Variable: `GITHUB_TOKEN`
  
- **GitLab CI/CD**
  - Environment Variable: `GITLAB_TOKEN`

### Container Registry
- **Docker Hub**
  - Environment Variables: `DOCKER_USERNAME`, `DOCKER_PASSWORD`
  
- **GitHub Container Registry**
  - Environment Variable: `GITHUB_TOKEN`

## 📋 Environment Configuration Template

Create a `.env` file with the following template:

```bash
# Application Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
LOG_LEVEL=info
LOG_FORMAT=json

# Database Configuration
MONGO_URI=mongodb://admin:password@localhost:27017/ai_government_consultant?authSource=admin
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=password

# Security
JWT_SECRET=your-super-secure-jwt-secret-change-in-production
ENCRYPTION_KEY=your-32-character-encryption-key

# LLM Configuration (Choose one)
LLM_PROVIDER=gemini
LLM_API_KEY=your-gemini-api-key
# OPENAI_API_KEY=your-openai-api-key
# ANTHROPIC_API_KEY=your-anthropic-api-key

# Email Service (Optional)
SENDGRID_API_KEY=your-sendgrid-api-key
FROM_EMAIL=noreply@yourgovernment.gov

# Document Processing (Optional)
ADOBE_PDF_API_KEY=your-adobe-pdf-api-key

# OAuth (Optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Monitoring (Optional)
SENTRY_DSN=your-sentry-dsn
NEW_RELIC_LICENSE_KEY=your-newrelic-key

# File Storage (Optional)
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key
AWS_S3_BUCKET=your-s3-bucket-name
AWS_REGION=us-east-1

# Search Enhancement (Optional)
PINECONE_API_KEY=your-pinecone-api-key
ELASTICSEARCH_URL=your-elasticsearch-url
```

## 🚀 Deployment Considerations

### Minimum Requirements (Development)
- **Required**: LLM API Key (Gemini/OpenAI)
- **Required**: MongoDB (local or Atlas free tier)
- **Required**: Redis (local or cloud free tier)
- **Estimated monthly cost**: $0-50

### Production Requirements
- **Required**: All minimum requirements
- **Recommended**: Email service (SendGrid/SES)
- **Recommended**: SSL certificate (Let's Encrypt/Cloudflare)
- **Recommended**: Monitoring (Sentry)
- **Recommended**: File storage (S3/GCS)
- **Estimated monthly cost**: $100-500 (depending on usage)

### Enterprise Requirements
- **Required**: All production requirements
- **Required**: Advanced monitoring (New Relic/DataDog)
- **Required**: Secrets management (Vault/AWS Secrets)
- **Required**: Vector database (Pinecone/Weaviate)
- **Required**: Advanced search (Elasticsearch)
- **Estimated monthly cost**: $500-2000+ (depending on scale)

## 📝 Setup Priority

### Phase 1 (MVP)
1. ✅ LLM API Key (Gemini recommended)
2. ✅ MongoDB connection
3. ✅ Redis connection
4. ✅ JWT secret generation

### Phase 2 (Production Ready)
1. Email service setup
2. SSL certificate configuration
3. Error monitoring (Sentry)
4. File storage (S3/GCS)

### Phase 3 (Enterprise)
1. Advanced monitoring
2. Vector database integration
3. Advanced search capabilities
4. Secrets management
5. Multi-region deployment

## 🔒 Security Notes

- **Never commit API keys to version control**
- Use environment variables or secrets management
- Rotate API keys regularly
- Set up API key usage alerts and limits
- Use least-privilege access principles
- Enable audit logging for all services
- Implement rate limiting and DDoS protection

## 📞 Support and Documentation

Each service provider offers comprehensive documentation and support:
- Most services provide free tiers for development
- Enterprise support is available for production deployments
- Community forums and documentation are available for all services
- Consider setting up monitoring and alerting for API usage and costs

This document should be updated as new services are integrated or requirements change.