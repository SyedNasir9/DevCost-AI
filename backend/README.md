# DevCost AI Backend

A production-grade Go backend for cloud cost optimization platform.

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis (optional for development)

### Development Setup

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start database**
   ```bash
   # Using Docker
   docker-compose up -d postgres
   
   # Or local PostgreSQL
   # Ensure database exists: createdb devcost_ai
   ```

4. **Run the server**
   ```bash
   go run cmd/server/main.go
   ```

5. **Build for production**
   ```bash
   go build -o build/devcost-ai cmd/server/main.go
   ./build/devcost-ai
   ```

## 📁 Project Structure

```
backend/
├── cmd/server/           # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── db/              # Database setup and connections
│   ├── handlers/        # HTTP request handlers
│   ├── models/          # Data models and entities
│   ├── router/          # HTTP routing and middleware
│   ├── repository/      # Data access layer (TODO)
│   ├── services/        # Business logic (TODO)
│   ├── middleware/      # HTTP middleware (TODO)
│   └── scheduler/       # Background jobs (TODO)
├── pkg/
│   └── logger/          # Structured logging utilities
├── migrations/          # Database migrations
├── Dockerfile          # Container configuration
└── Makefile           # Development automation
```

## 🔧 Configuration

The application uses environment variables for configuration. See `.env.example` for all available options.

Key configuration sections:

- **Database**: PostgreSQL connection settings
- **Server**: HTTP server configuration
- **Logging**: Structured logging with Zap
- **JWT**: Authentication token settings
- **Cloud**: AWS, GCP, Azure API credentials

## 🏥 Health Check

The server provides a comprehensive health check endpoint:

```bash
curl http://localhost:8080/health
```

Response includes:
- Service status
- Database connectivity
- Version information
- Timestamp

## 📊 API Documentation

Swagger documentation is available at:
```
http://localhost:8080/swagger/index.html
```

## 🛠️ Development Tools

### Makefile Commands

```bash
make build          # Build the application
make run            # Run the application
make test           # Run tests
make fmt            # Format code
make lint           # Run linter
make docker-build   # Build Docker image
make docker-up      # Start services with Docker Compose
```

### Hot Reload (Development)

For development with hot reload:

```bash
# Install Air
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

## 🗄️ Database

### Migrations

Database migrations are handled automatically on startup using GORM AutoMigrate.

### Schema

The database includes:
- `users` - User accounts and authentication
- `cloud_accounts` - Cloud provider account connections
- `cost_data` - Detailed cost information
- `recommendations` - Cost optimization suggestions
- `cost_alerts` - Alert configurations and notifications

## 🔒 Security Features

- Structured logging with request tracking
- CORS configuration
- Graceful shutdown handling
- Environment-based configuration
- Input validation (TODO)

## 📝 Logging

Uses Zap for structured logging. Log levels:
- `debug` - Detailed debugging information
- `info` - General information (default)
- `warn` - Warning messages
- `error` - Error messages

## 🐳 Docker

### Build Image

```bash
docker build -t devcost-ai:latest .
```

### Run with Docker Compose

```bash
docker-compose up -d
```

This starts:
- PostgreSQL database
- Redis cache
- Backend API server

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## 📈 Monitoring

### Health Checks

- `/health` - Service health status
- Database connectivity checks
- Graceful degradation on service failures

### Metrics (TODO)

- Prometheus metrics integration
- Request/response timing
- Error tracking

## 🔄 Next Steps

1. **Authentication & Authorization**
   - JWT middleware
   - Role-based access control

2. **API Endpoints**
   - User management
   - Cloud account integration
   - Cost data collection
   - Recommendations engine

3. **Background Jobs**
   - Cost data collection scheduler
   - Report generation
   - Alert processing

4. **Testing**
   - Unit tests for all components
   - Integration tests
   - API endpoint tests

5. **Monitoring & Observability**
   - Prometheus metrics
   - Distributed tracing
   - Error tracking (Sentry)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details
