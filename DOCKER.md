# DevCost AI Docker Setup

This guide covers the Docker setup for DevCost AI, including development and production configurations.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Backend      │    │   PostgreSQL    │    │     Redis      │
│   (Go API)    │◄──►│   Database     │◄──►│    Cache        │
│   Port: 8080   │    │   Port: 5432   │    │   Port: 6379   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 Quick Start

### Production Setup

1. **Clone and setup**
   ```bash
   git clone <repository-url>
   cd devcost-ai
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Start all services**
   ```bash
   docker-compose up -d
   ```

3. **Access services**
   - **API**: http://localhost:8080
   - **Health Check**: http://localhost:8080/health
   - **Database Admin** (optional): http://localhost:5050
     ```bash
     docker-compose --profile admin up -d
     ```

### Development Setup

1. **Start with hot reload**
   ```bash
   docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d
   ```

2. **Access development services**
   - **API**: http://localhost:8081
   - **Database**: localhost:5433
   - **Redis**: localhost:6380

## 📁 File Structure

```
devcost-ai/
├── docker-compose.yml           # Production configuration
├── docker-compose.dev.yml       # Development overrides
├── docker-compose.override.yml   # Local customizations
├── backend/
│   ├── Dockerfile             # Production build
│   ├── Dockerfile.dev         # Development with hot reload
│   └── .air.toml            # Air configuration
├── .env.example               # Environment template
└── .env                     # Your local configuration (gitignored)
```

## ⚙️ Configuration

### Environment Variables

#### Required Variables
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `JWT_SECRET` (use strong secret in production)

#### Optional Variables
- `REDIS_PASSWORD`, `LOG_LEVEL`, `GIN_MODE`
- Cloud provider credentials (AWS, GCP, Azure)

#### Development Overrides
- `GIN_MODE=debug`
- `LOG_LEVEL=debug`
- Custom ports to avoid conflicts

### Security Best Practices

1. **Never commit `.env` files** to version control
2. **Use strong JWT secrets** in production
3. **Enable SSL/TLS** in production environments
4. **Use different ports** for development
5. **Regularly update base images** for security patches

## 🐳 Docker Features

### Multi-Stage Build
- **Builder Stage**: Compiles Go binary with build caching
- **Runtime Stage**: Lightweight Alpine image
- **Security**: Non-root user, minimal attack surface
- **Optimization**: Static binary, reduced size

### Health Checks
- **PostgreSQL**: `pg_isready` command
- **Redis**: `redis-cli ping` command
- **Backend**: HTTP health endpoint
- **Auto-restart**: Services restart on failure

### Volume Management
- **PostgreSQL data**: Persistent across restarts
- **Redis data**: Cache persistence
- **Application logs**: Centralized logging
- **Migration files**: Database initialization

### Networking
- **Custom bridge network**: Service isolation
- **Service discovery**: Containers communicate by name
- **Port mapping**: Host access for development
- **Health dependencies**: Proper startup ordering

## 🔧 Development Workflow

### Local Development
```bash
# Start with hot reload
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# View logs
docker-compose logs -f backend

# Stop services
docker-compose down

# Rebuild with changes
docker-compose up --build
```

### Production Deployment
```bash
# Production deployment
docker-compose -f docker-compose.yml up -d

# Scale backend (if needed)
docker-compose up -d --scale backend=3

# Update without downtime
docker-compose up -d --no-deps backend
```

## 📊 Monitoring

### Health Status
```bash
# Check all services
docker-compose ps

# Check service health
docker-compose exec postgres pg_isready -U devcost -d devcost_ai
docker-compose exec redis redis-cli ping
curl http://localhost:8080/health
```

### Logs
```bash
# All service logs
docker-compose logs

# Specific service logs
docker-compose logs -f backend
docker-compose logs -f postgres

# Real-time logs with tail
docker-compose logs -f --tail=100 backend
```

## 🛠️ Troubleshooting

### Common Issues

1. **Port conflicts**
   ```bash
   # Check port usage
   netstat -tulpn | grep :8080
   
   # Change ports in .env
   SERVER_PORT=8081
   DB_PORT=5433
   ```

2. **Database connection issues**
   ```bash
   # Check database logs
   docker-compose logs postgres
   
   # Test connection manually
   docker-compose exec postgres psql -U devcost -d devcost_ai -c "SELECT 1;"
   ```

3. **Permission issues**
   ```bash
   # Fix volume permissions
   sudo chown -R $USER:$USER ./backend/logs
   
   # Clean up containers
   docker-compose down -v
   docker system prune -f
   ```

### Debug Mode

Enable detailed debugging:
```bash
# Set debug environment
echo "GIN_MODE=debug" >> .env
echo "LOG_LEVEL=debug" >> .env

# Restart with debug
docker-compose up -d
```

## 🚀 Production Considerations

### Security
- **Remove development tools**: Air, debug ports
- **Use HTTPS**: Terminate SSL at load balancer
- **Secret management**: Use Docker secrets or AWS KMS
- **Network policies**: Restrict inter-container communication

### Performance
- **Resource limits**: Set memory/CPU constraints
- **Connection pooling**: Configure appropriate pool sizes
- **Monitoring**: Enable metrics and health checks
- **Backup strategy**: Regular database backups

### Scaling
- **Load balancing**: Multiple backend instances
- **Database scaling**: Read replicas, connection pooling
- **Caching**: Redis for session and data caching
- **CDN**: Static asset delivery

## 🔄 CI/CD Integration

### GitHub Actions Example
```yaml
name: Deploy to Production
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Deploy to server
        run: |
          docker-compose -f docker-compose.yml up -d --build
          docker system prune -f
```

### Environment-Specific Configs
```bash
# Development
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# Staging
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# Production
docker-compose -f docker-compose.yml up -d
```

## 📚 Additional Resources

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [PostgreSQL Docker Hub](https://hub.docker.com/_/postgres/)
- [Redis Docker Hub](https://hub.docker.com/_/redis/)
- [Go Docker Best Practices](https://pkg.go.dev/github.com/docker-library/golang)

## 🆘 Support

For Docker-related issues:
1. Check container logs: `docker-compose logs <service>`
2. Verify environment variables: `docker-compose exec backend env`
3. Test database connectivity: `docker-compose exec postgres psql...`
4. Check network: `docker network ls` and `docker network inspect`
