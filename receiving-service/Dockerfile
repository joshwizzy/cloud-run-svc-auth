# Build stage
FROM golang:1.17 AS builder

# Set the working directory
WORKDIR /app

# Copy Go module files
COPY go.* ./

# Download dependencies
RUN go mod download

# Copy Go files
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Final stage
FROM alpine:3.15

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server /app/server

# Expose the listening port
EXPOSE 8080

# Run the server
CMD ["/app/server"]