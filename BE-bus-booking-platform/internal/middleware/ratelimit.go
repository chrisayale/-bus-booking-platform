package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a per-IP sliding-window rate limiter.
// It limits requests from a single IP to `maxRequests` per `window` duration.
type RateLimiter struct {
	mu          sync.Mutex
	visitors    map[string]*visitor
	maxRequests int
	window      time.Duration
}

type visitor struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter creates a rate limiter allowing `maxRequests` per `window`.
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors:    make(map[string]*visitor),
		maxRequests: maxRequests,
		window:      window,
	}
	// Periodic cleanup of stale entries every 10 minutes
	go rl.cleanup()
	return rl
}

// Middleware returns a Gin handler that limits requests by client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.RemoteIP()
		}

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		now := time.Now()

		if !exists || now.Sub(v.windowStart) > rl.window {
			// New window
			rl.visitors[ip] = &visitor{
				count:       1,
				windowStart: now,
			}
			rl.mu.Unlock()
			c.Next()
			return
		}

		if v.count >= rl.maxRequests {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests. Please try again later.",
			})
			return
		}

		v.count++
		rl.mu.Unlock()
		c.Next()
	}
}

// cleanup periodically removes stale entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.windowStart) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
