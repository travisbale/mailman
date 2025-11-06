# Stage 1: Build Go binary
FROM golang:1.24-alpine AS go-builder

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
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from go-builder
COPY --from=go-builder /build/bin/mailman .

# Create non-root user
RUN addgroup -g 1000 mailman && \
    adduser -D -u 1000 -G mailman mailman && \
    chown -R mailman:mailman /app

USER mailman

# Expose gRPC port
EXPOSE 50051

# Run the application
ENTRYPOINT ["./mailman"]
CMD ["start"]
