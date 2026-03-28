# Multi-stage Dockerfile for Go backend
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME=unknown
ARG BUILD_DIRTY=unknown

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildCommit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME} -X main.buildDirty=${BUILD_DIRTY}" \
    -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildCommit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME} -X main.buildDirty=${BUILD_DIRTY}" \
    -o migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildCommit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME} -X main.buildDirty=${BUILD_DIRTY}" \
    -o essential-config-seeder ./cmd/essential-config-seeder
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildCommit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME} -X main.buildDirty=${BUILD_DIRTY}" \
    -o email-template-seeder ./cmd/email-template-seeder

# Final stage
FROM alpine:latest

# Install runtime dependencies
# - git is required for repository branch listing operations.
RUN apk --no-cache add ca-certificates tzdata postgresql-client wget git

# Create app user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY --from=builder /app/essential-config-seeder .
COPY --from=builder /app/email-template-seeder .

# Copy migrations if needed
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/tekton ./tekton
COPY bootstrap ./bootstrap

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./server"]
