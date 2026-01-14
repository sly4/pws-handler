# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

# Install git (required for fetching dependencies)
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
# CGO_ENABLED=0 creates a statically linked binary (no external dependencies)
RUN CGO_ENABLED=0 GOOS=linux go build -o weather-server .

# Stage 2: Create the final minimal image
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/weather-server .

# Expose the port defined in your code defaults
EXPOSE 8080

# Run the binary
CMD ["./weather-server"]