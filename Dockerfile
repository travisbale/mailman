# Stage 1: Build Go binary
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build


# Stage 2: Runtime
FROM alpine:latest

# Add ca-certificates for HTTPS connections to SendGrid API and other runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy the binary from go-builder
COPY --from=go-builder /build/bin/mailman .

# Create non-root user
RUN addgroup -g 1000 mailman && \
    adduser -D -u 1000 -G mailman mailman && \
    chown -R mailman:mailman /app

USER mailman

# Expose ports
EXPOSE 8080 50051

# Health check using HTTP endpoint (checks database connectivity)
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the application
ENTRYPOINT ["./mailman"]
CMD ["start"]
