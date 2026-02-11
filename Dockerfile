# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application (if needed for examples)
RUN go build -o /dev/null ./...

# Test stage
FROM golang:1.21-alpine AS tester

WORKDIR /app

# Install git for go modules
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy all source code
COPY . .

# Run tests
CMD ["go", "test", "-v", "./..."]

# Final stage for running examples
FROM golang:1.21-alpine AS runner

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Default command
CMD ["go", "test", "-v", "./..."]
