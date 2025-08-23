# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (CGO disabled for pure Go SQLite)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create app directory and user
RUN addgroup -g 1001 -S app && \
    adduser -u 1001 -S app -G app

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Create necessary directories
RUN mkdir -p configs templates static submissions && \
    chown -R app:app /app

# Copy default templates and static files
COPY --chown=app:app templates/ ./templates/
COPY --chown=app:app static/ ./static/

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["./main"]