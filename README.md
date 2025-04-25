# Go Rate Limiter Demo

This project demonstrates various rate limiting algorithms implemented in Go for an HTTP server.

## Overview

The application runs a simple HTTP server with two endpoints:

*   `/unlimited`: An endpoint with no rate limiting applied.
*   `/limited`: An endpoint protected by a configurable rate limiter.

It showcases different rate limiting strategies that can be easily swapped.

## Implemented Rate Limiters

The following rate limiting algorithms are implemented in the `internal/ratelimiter` package:

1.  **Token Bucket:** (`TokenBucketRateLimiter`) Allows requests based on available tokens, which refill at a constant rate.
2.  **Fixed Window Counter:** (`FixedWindowRateLimiter`) Limits requests to a fixed number within discrete time windows.
3.  **Sliding Window Log:** (`SlidingWindowLogRateLimiter`) Limits requests based on the count of timestamps within a rolling time window. More accurate but potentially memory-intensive.
4.  **Sliding Window Counter:** (`SlidingWindowCounterRateLimiter`) Approximates the count within a rolling window using counters for the current and previous windows. A balance between accuracy and performance/memory.

## Project Structure

```
rate-limiter/
├── cmd/
│   └── app/
│       └── main.go        # Application entry point
├── internal/
│   ├── ratelimiter/
│   │   └── ratelimiter.go # Rate limiter interface and implementations
│   └── server/
│       └── server.go      # HTTP server setup and handlers
├── docker-compose.yml     # Docker Compose configuration
├── Dockerfile             # Dockerfile for building the application
├── go.mod                 # Go module configuration
└── README.md              # This file
```

## Getting Started

### Prerequisites

*   Go (version 1.18 or later recommended)

### Running the Application

1.  Clone the repository (if applicable).
2.  Navigate to the project root directory (`rate-limiter`).
3.  Run the application:
    ```bash
    go run ./cmd/app/main.go
    ```
4.  The server will start, typically on `127.0.0.1:8080`.
5.  You can access the endpoints using a web browser or `curl`:
    *   `curl http://127.0.0.1:8080/unlimited`
    *   `curl http://127.0.0.1:8080/limited`

### Configuring the Rate Limiter

To change the rate limiting algorithm or its parameters:

1.  Open `cmd/app/main.go`.
2.  Locate the section where the `rl` variable is initialized.
3.  Comment out the current limiter and uncomment/modify the desired one.

   ```go
   func main() {
       // Choose and initialize the desired rate limiter
       // Example: Token Bucket (10 tokens capacity, 1 token per second)
       // rl := ratelimiter.NewTokenBucketRateLimiter(10, time.Second)

       // Example: Fixed Window (10 requests per minute)
       // rl := ratelimiter.NewFixedWindowRateLimiter(1*time.Minute, 10)

       // Example: Sliding Window Log (15 requests per minute)
       // rl := ratelimiter.NewSlidingWindowLogRateLimiter(1*time.Minute, 15)

       // Example: Sliding Window Counter (6 requests per 60 seconds)
       rl := ratelimiter.NewSlidingWindowCounterRateLimiter(60*time.Second, 6)

       addr := "127.0.0.1:8080"
       log.Println("Initializing server...")
       server.StartServer(addr, rl)
   }
   ```
4.  Save the file and restart the application (`go run ./cmd/app/main.go`).

## Running with Docker

You can also run this application using Docker and Docker Compose.

**Prerequisites:**

*   [Docker](https://docs.docker.com/get-docker/)
*   [Docker Compose](https://docs.docker.com/compose/install/)

**Steps:**

1.  **Build the Docker image:**

    ```bash
    docker build -t rate-limiter-app .
    ```

2.  **Run using Docker Compose:**

    This command will start the rate limiter service in the background.

    ```bash
    docker-compose up -d
    ```

3.  **Access the service:**

    The service will be available on your host machine at port 8088.

    *   Limited endpoint: [http://localhost:8088/limited](http://localhost:8088/limited)
    *   Unlimited endpoint: [http://localhost:8088/unlimited](http://localhost:8088/unlimited)

4.  **Stop the service:**

    ```bash
    docker-compose down
    ```
