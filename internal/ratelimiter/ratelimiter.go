package ratelimiter

import (
	"sync"
	"time"
)

// RateLimiter interface defines the rate limiting behavior
type RateLimiter interface {
	Allow(ip string) bool
}

// --- Token Bucket Implementation ---

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
	capacity int // Store capacity for new buckets
	rate    time.Duration // Time between token additions
}

// NewTokenBucketRateLimiter creates a new TokenBucketRateLimiter
func NewTokenBucketRateLimiter(capacity int, rate time.Duration) *TokenBucketRateLimiter {
	rl := &TokenBucketRateLimiter{
		buckets: make(map[string]*TokenBucket),
		capacity: capacity,
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
			tokens:     rl.capacity, // Start with full bucket based on config
			capacity:   rl.capacity,
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

// cleanup removes buckets that haven't been used in a while
func (rl *TokenBucketRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.buckets {
			// Clean up if not refilled in 10 minutes
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed, consuming a token if possible
func (rl *TokenBucketRateLimiter) Allow(ip string) bool {
	bucket := rl.getBucket(ip)
	// Need lock here as refill and token decrement modify bucket state
	rl.mu.Lock()
	rl.refill(bucket)
	if bucket.tokens > 0 {
		bucket.tokens--
		rl.mu.Unlock()
		return true
	}
	rl.mu.Unlock()
	return false
}

// --- Fixed Window Counter Implementation ---

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
		now := time.Now()
		for ip, bucket := range rl.buckets {
			// Clean up if the window started more than 2 window sizes ago
			if now.Sub(bucket.windowStart) > 2*rl.windowSize {
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

// --- Sliding Window Log Implementation ---

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
		now := time.Now()
		for ip, bucket := range rl.buckets {
			// Clean up if not accessed in 2 window sizes
			if now.Sub(bucket.lastAccess) > 2*rl.windowSize {
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
	// Optimize timestamp removal: find the first index to keep
	validStartIndex := 0
	for i, ts := range bucket.timestamps {
		if !ts.Before(cutoff) {
			validStartIndex = i
			break
		}
		// If all timestamps are before cutoff, set index beyond slice
		if i == len(bucket.timestamps)-1 && ts.Before(cutoff) {
		    validStartIndex = len(bucket.timestamps)
		}
	}
	bucket.timestamps = bucket.timestamps[validStartIndex:]

	// Check if adding a new request would exceed the threshold
	if len(bucket.timestamps) < rl.threshold {
		bucket.timestamps = append(bucket.timestamps, now)
		return true
	}
	return false
}

// --- Sliding Window Counter Implementation ---

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
		now := time.Now()
		for ip, bucket := range rl.buckets {
			// Clean up if not accessed in 2 window sizes
			if now.Sub(bucket.lastAccess) > 2*rl.windowSize {
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
	currentWindowStart := now.Truncate(rl.windowSize)

	// If we've moved into a new window or further
	if !bucket.windowStart.Equal(currentWindowStart) {
		// Calculate how many windows have passed
		windowsPassed := int(currentWindowStart.Sub(bucket.windowStart) / rl.windowSize)
		if windowsPassed == 1 {
			// Moved to the immediately next window
			bucket.previousCount = bucket.currentCount
		} else {
			// Skipped one or more windows, previous count is 0
			bucket.previousCount = 0
		}
		bucket.currentCount = 0
		bucket.windowStart = currentWindowStart
	}

	// Calculate approximate count in the sliding window
	elapsedInCurrentWindow := now.Sub(bucket.windowStart)
	weightPrevious := (float64(rl.windowSize) - float64(elapsedInCurrentWindow)) / float64(rl.windowSize)
	estimatedCount := float64(bucket.previousCount)*weightPrevious + float64(bucket.currentCount)

	// Check if adding a request would exceed the threshold
	if estimatedCount < float64(rl.threshold) {
		bucket.currentCount++
		return true
	}
	return false
}

// --- Helper Functions ---

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
