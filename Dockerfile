# ============================================
# AI Gateway - Multi-stage Docker Build
# Optimized for small image size and security
# ============================================

# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build the binary with optimizations
# CGO_ENABLED=1 for SQLite support
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o /build/ai-gateway \
    ./cmd/gateway

# Stage 2: Minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl sqlite-libs

# Create non-root user for security
RUN addgroup -g 1000 gateway && \
    adduser -u 1000 -G gateway -s /bin/sh -D gateway

WORKDIR /app

# Create necessary directories
RUN mkdir -p /app/configs /app/data && \
    chown -R gateway:gateway /app

# Copy binary from builder
COPY --from=builder /build/ai-gateway /app/ai-gateway

# Copy default config
COPY configs/config.json /app/configs/config.json

# Switch to non-root user
USER gateway

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV GIN_MODE=release \
    CONFIG_PATH=/app/configs/config.json

# Run the binary
ENTRYPOINT ["/app/ai-gateway"]
