# --- Build Stage ---
# Use an official Golang runtime as a parent image
FROM golang:1.24.2-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files to download dependencies first
COPY go.mod  ./

# Copy the source code into the container
COPY . .

# Build the Go app statically
# -ldflags "-w -s" reduces the binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/app/main.go

# --- Runtime Stage ---
# Use a minimal non-distro image based on Debian
FROM gcr.io/distroless/static-debian11

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/server /app/server

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/app/server"]
