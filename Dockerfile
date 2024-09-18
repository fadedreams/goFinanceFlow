# Use a specific version of Go
FROM golang:latest AS builder

# Set the working directory
WORKDIR /app


ENV GOPROXY=https://goproxy.io,direct
COPY go.mod go.sum ./


# Copy go mod and sum files
# Download dependencies with retry logic
# RUN --mount=type=cache,target=/go/pkg/mod \
#     --mount=type=cache,target=/root/.cache/go-build \
#     go mod download || (sleep 5 && go mod download) || (sleep 10 && go mod download)
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /myapp

# Use a small alpine image for the final stage
FROM alpine:latest

# Copy the binary from the builder stage
COPY --from=builder /myapp /myapp

# Expose both HTTP and gRPC ports
EXPOSE 8080 9090

# Run the binary
CMD ["/myapp"]
