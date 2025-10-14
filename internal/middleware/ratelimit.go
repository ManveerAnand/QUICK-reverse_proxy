package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a middleware that limits the number of requests a client can make.
type RateLimiter struct {
	rate  int           // Maximum number of requests allowed
	burst int           // Maximum burst size
	limit map[string]*time.Ticker
	mu    sync.Mutex
}

// NewRateLimiter creates a new RateLimiter with the specified rate and burst.
func NewRateLimiter(rate int, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  rate,
		burst: burst,
		limit: make(map[string]*time.Ticker),
	}
}

// ServeHTTP implements the http.Handler interface for RateLimiter.
func (rl *RateLimiter) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	clientIP := r.RemoteAddr

	rl.mu.Lock()
	ticker, exists := rl.limit[clientIP]
	if !exists {
		ticker = time.NewTicker(time.Second / time.Duration(rl.rate))
		rl.limit[clientIP] = ticker
	}
	rl.mu.Unlock()

	select {
	case <-ticker.C:
		next(w, r)
	default:
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	}
}

// Cleanup stops the tickers for all clients.
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for _, ticker := range rl.limit {
		ticker.Stop()
	}
	rl.limit = make(map[string]*time.Ticker)
}