# Use an official Go runtime as the base image
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project and build it
COPY . .
RUN go build -o crawler ./main.go

# Use a minimal image for the final runtime
FROM alpine:latest

# Set working directory
WORKDIR /root/

# Copy the built binary from the builder
COPY --from=builder /app/crawler .

# Set environment variables
ENV PORT=3005
ENV GRPC_PORT=50051

# Expose HTTP and gRPC ports
EXPOSE 3005 50051

# Start the application
CMD ["./crawler"]