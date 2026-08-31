package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

type clientBucket struct {
	tokens     int
	lastRefill time.Time
}

// RateLimiter creates an in-memory token bucket rate limiter middleware.
func RateLimiter(maxRequests int, window time.Duration) fiber.Handler {
	var mu sync.Mutex
	buckets := make(map[string]*clientBucket)

	// Clean up stale buckets every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for k, v := range buckets {
				if now.Sub(v.lastRefill) > window*2 {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		// Identify client by UserID if authenticated, otherwise IP
		key := c.IP()
		if sess := SessionFromCtx(c); sess != nil {
			key = "usr_" + strconv.FormatInt(sess.UserID, 10)
		}

		mu.Lock()
		b, exists := buckets[key]
		now := time.Now()

		if !exists {
			b = &clientBucket{
				tokens:     maxRequests - 1,
				lastRefill: now,
			}
			buckets[key] = b
			mu.Unlock()
			c.Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
			c.Set("X-RateLimit-Remaining", strconv.Itoa(b.tokens))
			return c.Next()
		}

		// Refill tokens based on elapsed time
		elapsed := now.Sub(b.lastRefill)
		if elapsed >= window {
			b.tokens = maxRequests
			b.lastRefill = now
		}

		if b.tokens <= 0 {
			mu.Unlock()
			c.Set("Retry-After", strconv.Itoa(int(window.Seconds())))
			return response.Error(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded. Please retry shortly.")
		}

		b.tokens--
		remaining := b.tokens
		mu.Unlock()

		c.Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		return c.Next()
	}
}
