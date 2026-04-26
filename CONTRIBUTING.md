# Contributing to DevCost AI

Thank you for your interest in contributing to DevCost AI! This document provides guidelines for contributing to the project.

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- Git

### Setting Up Development Environment

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/DevCost-AI.git
   cd DevCost-AI
   ```

2. **Start development services**
   ```bash
   # Using Docker (recommended)
   docker compose -f docker-compose.dev.yml up -d

   # Or start individual services
   cd backend && go run cmd/server/main.go
   cd frontend && npm run dev
   ```

3. **Verify setup**
   ```bash
   # Backend health check
   curl http://localhost:8080/health

   # Frontend
   open http://localhost:3000
   ```

## 📝 Code Guidelines

### Backend (Go)

- Follow standard Go conventions (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Write tests for new functionality

```go
// Good
func (s *WasteDetectionService) DetectIdleEC2(ctx context.Context) ([]WasteResult, error) {
    // Implementation
}

// Bad
func (s *WasteDetectionService) d(c context.Context) ([]WasteResult, error) {
    // Implementation
}
```

### Frontend (TypeScript)

- Use TypeScript for all new code
- Follow React best practices (hooks, functional components)
- Use Tailwind CSS for styling
- Keep components small and focused

```typescript
// Good
export function ResourceCard({ resource }: { resource: Resource }) {
  return (
    <div className="rounded-lg border p-4">
      <h3>{resource.name}</h3>
    </div>
  );
}
```

### Commit Messages

Use conventional commit format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(backend): add waste detection for Lambda functions
fix(frontend): resolve dashboard loading state
docs: update API documentation
```

## 🔀 Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

2. **Make your changes**
   - Write clear, documented code
   - Add or update tests
   - Update documentation if needed

3. **Test locally**
   ```bash
   # Backend tests
   cd backend && go test ./...

   # Frontend tests
   cd frontend && npm test
   ```

4. **Push and create PR**
   ```bash
   git push origin feature/amazing-feature
   ```
   Then open a Pull Request on GitHub.

5. **PR Requirements**
   - Clear description of changes
   - All tests passing
   - No merge conflicts
   - Follows code style guidelines

## 🐛 Reporting Issues

### Bug Reports

Include:
- Clear description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Docker version, etc.)
- Relevant logs or screenshots

### Feature Requests

Include:
- Clear description of the feature
- Use case / problem it solves
- Proposed solution (if any)
- Alternatives considered

## 📂 Project Structure

```
DevCost-AI/
├── backend/
│   ├── cmd/server/          # Entry point
│   ├── internal/
│   │   ├── config/          # Configuration
│   │   ├── db/              # Database
│   │   ├── handlers/        # HTTP handlers
│   │   ├── models/          # Data models
│   │   ├── repositories/    # Data access
│   │   ├── services/        # Business logic
│   │   └── router/          # Route setup
│   └── migrations/          # DB migrations
├── frontend/
│   ├── src/
│   │   ├── app/            # Next.js pages
│   │   └── lib/            # Utilities
│   └── public/             # Static assets
└── docker-compose.yml
```

## 🧪 Testing

### Backend Testing

```bash
cd backend

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/services/...
```

### Frontend Testing

```bash
cd frontend

# Run tests
npm test

# Run with coverage
npm test -- --coverage
```

## 📚 Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update API documentation for endpoint changes
- Add examples for new features

## 🎯 Priority Areas

We especially welcome contributions in:

1. **Cloud Provider Support**
   - GCP integration
   - Azure integration
   - Additional AWS services

2. **Cost Detection Rules**
   - New waste patterns
   - Improved thresholds
   - Custom rule configuration

3. **UI/UX Improvements**
   - Better visualizations
   - Mobile responsiveness
   - Accessibility

4. **Testing**
   - Unit tests
   - Integration tests
   - E2E tests

## ❓ Questions?

- Open a GitHub Discussion
- Check existing issues
- Review the README

Thank you for contributing! 🙏
