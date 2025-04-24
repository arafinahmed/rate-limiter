package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// RateLimiter interface defines the rate limiting behavior
type RateLimiter interface {
	Allow(ip string) bool
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     int       // Current number of tokens
	capacity   int       // Maximum number of tokens
	lastRefill time.Time // Last time tokens were refilled
}

// TokenBucketRateLimiter manages token buckets for multiple clients
type TokenBucketRateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.Mutex
	rate    time.Duration // Time between token additions
}

// NewTokenBucketRateLimiter creates a new TokenBucketRateLimiter
func NewTokenBucketRateLimiter(capacity int, rate time.Duration) *TokenBucketRateLimiter {
	rl := &TokenBucketRateLimiter{
		buckets: make(map[string]*TokenBucket),
		rate:    rate,
	}
	go rl.cleanup()
	return rl
}

// getBucket retrieves or creates a token bucket for an IP
func (rl *TokenBucketRateLimiter) getBucket(ip string) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &TokenBucket{
			tokens:     10, // Start with full bucket
			capacity:   10,
			lastRefill: time.Now(),
		}
		rl.buckets[ip] = bucket
	}
	return bucket
}

// refill adds tokens to a bucket based on elapsed time
func (rl *TokenBucketRateLimiter) refill(bucket *TokenBucket) {
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / rl.rate)
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.capacity, bucket.tokens+tokensToAdd)
		bucket.lastRefill = now
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// cleanup removes buckets that haven't been used in a while
func (rl *TokenBucketRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			if time.Since(bucket.lastRefill) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed, consuming a token if possible
func (rl *TokenBucketRateLimiter) Allow(ip string) bool {
	bucket := rl.getBucket(ip)
	rl.refill(bucket)
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}

// FixedWindowCounter represents a counter for a fixed time window
type FixedWindowCounter struct {
	count      int       // Number of requests in the current window
	windowStart time.Time // Start of the current window
}

// FixedWindowRateLimiter manages fixed window counters for multiple clients
type FixedWindowRateLimiter struct {
	buckets     map[string]*FixedWindowCounter
	mu          sync.Mutex
	windowSize  time.Duration // Size of the time window
	threshold   int           // Maximum requests per window
}

// NewFixedWindowRateLimiter creates a new FixedWindowRateLimiter
func NewFixedWindowRateLimiter(windowSize time.Duration, threshold int) *FixedWindowRateLimiter {
	rl := &FixedWindowRateLimiter{
		buckets:    make(map[string]*FixedWindowCounter),
		windowSize: windowSize,
		threshold:  threshold,
	}
	go rl.cleanup()
	return rl
}

// getBucket retrieves or creates a counter for an IP
func (rl *FixedWindowRateLimiter) getBucket(ip string) *FixedWindowCounter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &FixedWindowCounter{
			count:      0,
			windowStart: time.Now().Truncate(rl.windowSize),
		}
		rl.buckets[ip] = bucket
	}
	return bucket
}

// cleanup removes buckets that haven't been used in a while
func (rl *FixedWindowRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			if time.Since(bucket.windowStart) > 2*rl.windowSize {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed, incrementing the counter if possible
func (rl *FixedWindowRateLimiter) Allow(ip string) bool {
	bucket := rl.getBucket(ip)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	currentWindow := now.Truncate(rl.windowSize)

	// If we're in a new window, reset the counter
	if !bucket.windowStart.Equal(currentWindow) {
		bucket.count = 0
		bucket.windowStart = currentWindow
	}

	// Check if we can allow the request
	if bucket.count < rl.threshold {
		bucket.count++
		return true
	}
	return false
}

// SlidingWindowLog represents a log of request timestamps
type SlidingWindowLog struct {
	timestamps []time.Time // Sorted list of request timestamps
	lastAccess time.Time   // Last time this log was accessed
}

// SlidingWindowLogRateLimiter manages sliding window logs for multiple clients
type SlidingWindowLogRateLimiter struct {
	buckets     map[string]*SlidingWindowLog
	mu          sync.Mutex
	windowSize  time.Duration // Size of the sliding window
	threshold   int           // Maximum requests per window
}

// NewSlidingWindowLogRateLimiter creates a new SlidingWindowLogRateLimiter
func NewSlidingWindowLogRateLimiter(windowSize time.Duration, threshold int) *SlidingWindowLogRateLimiter {
	rl := &SlidingWindowLogRateLimiter{
		buckets:    make(map[string]*SlidingWindowLog),
		windowSize: windowSize,
		threshold:  threshold,
	}
	go rl.cleanup()
	return rl
}

// getBucket retrieves or creates a log for an IP
func (rl *SlidingWindowLogRateLimiter) getBucket(ip string) *SlidingWindowLog {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &SlidingWindowLog{
			timestamps: []time.Time{},
			lastAccess: time.Now(),
		}
		rl.buckets[ip] = bucket
	}
	return bucket
}

// cleanup removes buckets that haven't been used in a while
func (rl *SlidingWindowLogRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			if time.Since(bucket.lastAccess) > 2*rl.windowSize {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed, adding a timestamp if possible
func (rl *SlidingWindowLogRateLimiter) Allow(ip string) bool {
	bucket := rl.getBucket(ip)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket.lastAccess = now

	// Remove timestamps older than the window
	cutoff := now.Add(-rl.windowSize)
	newTimestamps := make([]time.Time, 0, len(bucket.timestamps))
	for _, ts := range bucket.timestamps {
		if !ts.Before(cutoff) {
			newTimestamps = append(newTimestamps, ts)
		}
	}
	bucket.timestamps = newTimestamps

	// Check if adding a new request would exceed the threshold
	if len(bucket.timestamps) < rl.threshold {
		bucket.timestamps = append(bucket.timestamps, now)
		return true
	}
	return false
}

// SlidingWindowCounter represents counters for current and previous windows
type SlidingWindowCounter struct {
	currentCount  int       // Requests in the current window
	previousCount int       // Requests in the previous window
	windowStart   time.Time // Start of the current window
	lastAccess    time.Time // Last time this counter was accessed
}

// SlidingWindowCounterRateLimiter manages sliding window counters
type SlidingWindowCounterRateLimiter struct {
	buckets     map[string]*SlidingWindowCounter
	mu          sync.Mutex
	windowSize  time.Duration // Size of the window
	threshold   int           // Maximum requests per window
}

// NewSlidingWindowCounterRateLimiter creates a new SlidingWindowCounterRateLimiter
func NewSlidingWindowCounterRateLimiter(windowSize time.Duration, threshold int) *SlidingWindowCounterRateLimiter {
	rl := &SlidingWindowCounterRateLimiter{
		buckets:    make(map[string]*SlidingWindowCounter),
		windowSize: windowSize,
		threshold:  threshold,
	}
	go rl.cleanup()
	return rl
}

// getBucket retrieves or creates a counter for an IP
func (rl *SlidingWindowCounterRateLimiter) getBucket(ip string) *SlidingWindowCounter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &SlidingWindowCounter{
			currentCount:  0,
			previousCount: 0,
			windowStart:   time.Now().Truncate(rl.windowSize),
			lastAccess:    time.Now(),
		}
		rl.buckets[ip] = bucket
	}
	return bucket
}

// cleanup removes buckets that haven't been used in a while
func (rl *SlidingWindowCounterRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			if time.Since(bucket.lastAccess) > 2*rl.windowSize {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed, incrementing the counter if possible
func (rl *SlidingWindowCounterRateLimiter) Allow(ip string) bool {
	bucket := rl.getBucket(ip)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket.lastAccess = now
	currentWindow := now.Truncate(rl.windowSize)

	// If we've moved to a new window, shift counts
	if !bucket.windowStart.Equal(currentWindow) {
		bucket.previousCount = bucket.currentCount
		bucket.currentCount = 0
		bucket.windowStart = currentWindow
	}

	// Calculate weighted count
	elapsed := now.Sub(bucket.windowStart).Seconds()
	weight := (float64(rl.windowSize)/float64(time.Second) - elapsed) / (float64(rl.windowSize) / float64(time.Second))
	weightedCount := float64(bucket.currentCount) + float64(bucket.previousCount)*weight

	// Check if adding a request would exceed the threshold
	if weightedCount < float64(rl.threshold) {
		bucket.currentCount++
		return true
	}
	return false
}

func main() {
	// Initialize sliding window counter rate limiter: 60s window, 60 requests threshold
	rl := NewSlidingWindowCounterRateLimiter(60*time.Second, 6)

	// Define handlers
	http.HandleFunc("/unlimited", unlimitedHandler)
	http.HandleFunc("/limited", limitedHandler(rl))

	// Start the server
	addr := "127.0.0.1:8080"
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
func limitedHandler(rl RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP (simplified; in production, handle proxies)
		ip := r.RemoteAddr
		if rl.Allow(ip) {
			fmt.Fprint(w, "Limited, don't over use me!")
		} else {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		}
	}
}