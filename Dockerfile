FROM golang:1.24-alpine AS builder

ARG MAIN_PATH

WORKDIR /app

RUN apk update \
    && apk add --no-cache ca-certificates \
    && update-ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -o server $MAIN_PATH

FROM alpine:latest

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/server .

# Expose port
EXPOSE 4444

# Run the application
CMD ["./server"]
