# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies in a single layer
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies (this layer will be cached unless go.mod/go.sum changes)
RUN go mod download && go mod verify

# Copy only necessary source files (not the entire codebase)
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY configs/ ./configs/

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/server

# Final stage - use distroless for smaller, more secure image
FROM gcr.io/distroless/static-debian11:nonroot

# Copy the binary from builder stage
COPY --from=builder /app/main /app/main

# Copy configuration files
COPY --from=builder /app/configs /app/configs

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Run the application as non-root user (distroless nonroot user)
CMD ["./main"]