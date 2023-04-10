# Dockerfile

# Use the official Golang image as the base image
FROM golang:1.17 as builder

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o RedactedHook .

# Use a minimal Alpine image to run the binary
FROM alpine:latest

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder image
COPY --from=builder /app/RedactedHook .

# Expose the port
EXPOSE 42135

# Command to run the binary
CMD ["./RedactedHook"]
