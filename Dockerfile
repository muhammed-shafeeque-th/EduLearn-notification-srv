# -------- Stage 1: Build the Go binary --------
# Switched to Debian-based Go image to bypass Alpine emulation bugs.
# Git and ca-certificates are already installed in this image by default.
FROM golang:1.24-bookworm AS builder

# Enable Go modules and disable CGO for a statically linked binary
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set the working directory
WORKDIR /app

# Copy go mod files and download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
RUN go build -o notification ./cmd/server/main.go


# -------- Stage 2: Minimal runtime image --------
FROM gcr.io/distroless/base-debian12:nonroot

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/notification /app/

# Copy template files
COPY --from=builder /app/pkg/templates /app/pkg/templates/

# Expose service port
EXPOSE 50052

# Run the binary
ENTRYPOINT ["/app/notification"]