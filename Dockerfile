# Use Go runtime to build the binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install required dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum first (for caching dependencies)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -o crawler .

# Use a minimal runtime image for production
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy binary from the builder stage
COPY --from=builder /app/crawler .

# Ensure the binary is executable
RUN chmod +x /app/crawler

# Set environment variables
ENV PORT=3005
ENV GRPC_PORT=50051

# Expose ports
EXPOSE 3005 50051

# Run the application
CMD ["/app/crawler"]