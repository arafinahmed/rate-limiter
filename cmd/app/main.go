package main

import (
	"log"
	"time"

	"rate-limiter/internal/ratelimiter"
	"rate-limiter/internal/server"
)

func main() {
	// Choose and initialize the desired rate limiter
	// Example: Token Bucket (10 tokens capacity, 1 token per second)
	// rl := ratelimiter.NewTokenBucketRateLimiter(10, time.Second)

	// Example: Fixed Window (10 requests per minute)
	// rl := ratelimiter.NewFixedWindowRateLimiter(1*time.Minute, 10)

	// Example: Sliding Window Log (15 requests per minute)
	// rl := ratelimiter.NewSlidingWindowLogRateLimiter(1*time.Minute, 15)

	// Example: Sliding Window Counter (6 requests per 60 seconds - as before)
	rl := ratelimiter.NewSlidingWindowCounterRateLimiter(60*time.Second, 6)

	// Define server address
	addr := ":8080"

	// Start the server
	log.Println("Initializing server...")
	server.StartServer(addr, rl)
}
