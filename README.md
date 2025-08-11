# AI Government Consultant

An AI-powered government consulting platform that provides expert advice and guidance to government agencies and organizations. The system leverages AI to analyze documents and provide recommendations on policy development, strategic planning, operational efficiency, and technology implementation.

## Features

- **Document Processing**: Upload and analyze policy documents with AI-powered insights
- **AI Consultation**: Get expert recommendations on strategy, operations, and technology
- **Knowledge Management**: Maintain a searchable knowledge base of consultations and decisions
- **Audit & Compliance**: Comprehensive audit trails and transparency for all recommendations
- **Security**: Government-grade security with encryption and access controls

## Architecture

The platform is built with:
- **Backend**: Go with Gin framework
- **Database**: MongoDB with vector search capabilities
- **Cache**: Redis for session management and caching
- **AI**: Integration with large language models (Gemini)
- **Security**: JWT authentication, RBAC, and data encryption

## Quick Start

### Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- MongoDB 7.0+
- Redis 7.2+

### Development Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd ai-government-consultant
   ```

2. **Copy environment configuration**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration values
   ```

3. **Start development environment**
   ```bash
   make dev
   ```

   This will start:
   - Application server on http://localhost:8080
   - MongoDB on http://localhost:27017
   - Redis on http://localhost:6379
   - MongoDB Express on http://localhost:8081 (admin/admin)
   - Redis Commander on http://localhost:8082

4. **Test the application**
   ```bash
   curl http://localhost:8080/health
   ```

### Production Setup

1. **Build and run with Docker Compose**
   ```bash
   make prod-up
   ```

2. **Or build and run locally**
   ```bash
   make build
   ./bin/ai-government-consultant
   ```

## API Endpoints

### Health Check
- `GET /health` - Application health status
- `GET /api/v1/status` - Service status

### Authentication (Coming in Task 3)
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/logout` - User logout
- `POST /api/v1/auth/refresh` - Refresh token

### Documents (Coming in Task 4)
- `POST /api/v1/documents` - Upload document
- `GET /api/v1/documents` - List documents
- `GET /api/v1/documents/:id` - Get document details

### Consultations (Coming in Task 7)
- `POST /api/v1/consultations` - Create consultation
- `GET /api/v1/consultations` - List consultations
- `GET /api/v1/consultations/:id` - Get consultation details

## Configuration

The application can be configured through environment variables or YAML files:

### Environment Variables
See `.env.example` for all available configuration options.

### Configuration Files
- `configs/app.yaml` - Default application configuration
- Environment variables override YAML configuration

## Development

### Available Make Commands

```bash
make help          # Show all available commands
make dev           # Start development environment
make test          # Run tests
make build         # Build application
make fmt           # Format code
make lint          # Run linter
make clean         # Clean build artifacts
```

### Project Structure

```
.
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   └── server/          # HTTP server setup
├── pkg/
│   └── logger/          # Logging utilities
├── configs/             # Configuration files
├── scripts/             # Database and deployment scripts
├── docs/                # Documentation
└── docker-compose.yml   # Docker services
```

### Hot Reload Development

The development environment uses Air for hot reload:
```bash
make dev
```

Any changes to Go files will automatically rebuild and restart the application.

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race
```

## Security

This application implements government-grade security:

- **Authentication**: Multi-factor authentication with JWT tokens
- **Authorization**: Role-based access control (RBAC)
- **Encryption**: AES-256 encryption at rest and TLS 1.3 in transit
- **Audit Logging**: Comprehensive audit trails for all operations
- **Data Classification**: Automatic security classification and handling

## Deployment

### Docker Deployment
```bash
# Production deployment
make prod-up

# Development deployment with tools
make dev
```

### Environment Configuration
Ensure all required environment variables are set:
- `JWT_SECRET` - JWT signing secret
- `LLM_API_KEY` - AI service API key
- `MONGO_URI` - MongoDB connection string
- `REDIS_HOST` - Redis host

## Contributing

1. Follow Go coding standards and run `make fmt` before committing
2. Write tests for new functionality
3. Update documentation for API changes
4. Ensure all tests pass with `make test`

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions, please refer to the project documentation or create an issue in the repository.