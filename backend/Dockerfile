# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the executable
RUN go build -o /chimera-server ./cmd/server

# Stage 2: Create the final, minimal image
FROM alpine:3.18

WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /chimera-server .

# Expose the port the Go server listens on (e.g., 8080)
EXPOSE 8080

# Command to run the executable
CMD ["./chimera-server"]
