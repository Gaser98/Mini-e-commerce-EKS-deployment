# ----------------------------
# Build stage
# ----------------------------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git (needed for go mod)
RUN apk add --no-cache git

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o api ./cmd/api

# ----------------------------
# Runtime stage
# ----------------------------
FROM alpine:3.19

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api .

# Copy frontend files
COPY frontend/ ./frontend/

# Expose API port
EXPOSE 8080

# Run API
CMD ["./api"]
