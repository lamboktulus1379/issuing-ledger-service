# Stage 1: Build the Go application
FROM golang:1.23.3 AS builder

RUN apt-get update && \
        apt-get install -y --no-install-recommends tzdata
# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY constant/ ./constant/
COPY domain/ ./domain/
COPY infrastructure/ ./infrastructure/
COPY interfaces/ ./interfaces/
COPY usecase/ ./usecase/
COPY mocks/ ./mocks/
COPY database.go ./database.go
COPY config.json ./config.json
COPY goroutine.go ./goroutine.go
COPY router.go ./router.go
COPY main.go ./main.go
COPY go.mod ./go.mod
COPY go.sum ./go.sum

# Build the Go app
RUN GOOS=linux GOARCH=amd64 go build -o main .

# Stage 2: Create a minimal image with the compiled binary
FROM debian:bullseye-slim
# Create a non-root user and group, and install necessary packages including tzdata
# Install necessary packages including tzdata
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

# Set the timezone environment variable
ENV TZ=Asia/Jakarta

# Install necessary packages (if any)
#RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main /main
COPY --from=builder /app/config.json /config.json

# Change ownership of the copied files to the non-root user
RUN mkdir "logs" \
   && chown -R appuser /main /config.json /logs

# Switch to the non-root user
USER appuser

# Command to run the executable
CMD ["/main"]