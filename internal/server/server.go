package server

import (
	"fmt"
	"log"
	"net/http"
	"rate-limiter/internal/ratelimiter" // Import the internal ratelimiter package
)

// StartServer configures and starts the HTTP server
func StartServer(addr string, rl ratelimiter.RateLimiter) {
	// Define handlers
	http.HandleFunc("/unlimited", unlimitedHandler)
	http.HandleFunc("/limited", limitedHandler(rl))

	// Start the server
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// unlimitedHandler handles requests to /unlimited
func unlimitedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Unlimited! Let's Go!")
}

// limitedHandler creates a handler for /limited with rate limiting
func limitedHandler(rl ratelimiter.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP (simplified; in production, handle proxies, X-Forwarded-For)
		// For simplicity, using RemoteAddr. Might need refinement.
		ip := r.RemoteAddr
		// Consider splitting host and port if needed, e.g., ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		if rl.Allow(ip) {
			fmt.Fprint(w, "Limited, don't over use me!")
		} else {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		}
	}
}
