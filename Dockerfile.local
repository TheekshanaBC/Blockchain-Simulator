# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
# Only copy go.sum if it exists
COPY go.sum* ./
RUN go mod download

# Copy the entire project
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/valenced ./cmd/valenced

# Runtime stage
FROM alpine:latest

# Add ca-certificates in case the node needs to make HTTPS requests to peers (e.g., railway app)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/valenced /usr/local/bin/valenced

# Default port, though docker-compose will override it
EXPOSE 8080

# Run the node
ENTRYPOINT ["valenced"]
