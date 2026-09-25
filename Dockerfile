# syntax=docker/dockerfile:1
# Stage 1: Build the Go application
FROM golang:1.27 AS builder

WORKDIR /app

# 1. Copy ONLY dependency manifests first
COPY go.mod go.sum ./

# 2. Download dependencies using BuildKit cache mount
# This layer will be instantly cached unless you modify go.mod
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 3. Copy the rest of the source code
COPY . .

# 4. Build the application using both module and build caches
# CGO_ENABLED=0 ensures a static binary for the runtime stage
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o main .

# Stage 2: Create a minimal image with the compiled binary
# Consider using debian:bookworm-slim as bullseye is aging
FROM debian:bullseye-slim

# Set the timezone environment variable
ENV TZ=Asia/Jakarta

# Install required runtime dependencies and clean up apt cache
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user and group
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Copy the binary and config from the builder stage
COPY --from=builder /app/main /main
COPY --from=builder /app/config.json /config.json

# Create log directory and change ownership
RUN mkdir /logs && chown -R appuser:appgroup /main /config.json /logs

# Switch to the non-root user
USER appuser

# Command to run the executable
CMD ["/main"]