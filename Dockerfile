FROM golang:alpine AS builder

ARG MAIN_PATH

WORKDIR /app

# Copy go mod and sum files
RUN go env -w GOCACHE=/go-cache
RUN go env -w GOMODCACHE=/gomod-cache
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/gomod-cache \
    go mod download

# Copy source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache \
    CGO_ENABLED=0 GOOS=linux go build -a -o server $MAIN_PATH

FROM alpine AS certs
RUN apk add -U --no-cache ca-certificates

FROM scratch

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/server .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose port
EXPOSE 4444

# Run the application
CMD ["./server"]
