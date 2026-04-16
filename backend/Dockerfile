# Multi-stage build for DevCost AI Backend
# Stage 1: Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies (will be cached if go.mod/go.sum don't change)
RUN go mod download

# Copy source code
COPY . .

# Build the application
# -w -s flags reduce binary size
# CGO_ENABLED=0 creates static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main cmd/server/main.go

# Stage 2: Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1001 -S devcost && \
    adduser -u 1001 -S devcost -G devcost

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy migration files
COPY --from=builder /app/migrations ./migrations

# Create necessary directories
RUN mkdir -p /app/logs /app/tmp

# Change ownership to non-root user
RUN chown -R devcost:devcost /app

# Switch to non-root user
USER devcost

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV SERVER_PORT=8080

# Run the application
CMD ["./main"]
