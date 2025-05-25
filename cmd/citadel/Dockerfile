FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -o citadel ./cmd/citadel

FROM alpine:latest

# Install dependencies for SQLite
RUN apk add --no-cache ca-certificates libc6-compat sqlite

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/citadel .

# Create data directory
RUN mkdir -p /data

# Expose port
EXPOSE 4444

# Run the application
CMD ["./citadel"]
