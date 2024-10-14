# Start from the official Go image
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o feeder-sync ./cmd/webserver
# Build the migrator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o feeder-sync-migrate ./cmd/db_migrator

# Start a new stage from scratch
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/feeder-sync .
COPY --from=builder /app/feeder-sync-migrate .

# Expose port 34217
EXPOSE 34217

# Command to run the executable
CMD ["./feeder-sync"]
