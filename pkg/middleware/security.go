package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders applies production HTTP security headers to all incoming requests.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevent MIME-type sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// Protect against clickjacking
		c.Set("X-Frame-Options", "SAMEORIGIN")

		// Enable XSS protection filter
		c.Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information sent in requests
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features and APIs
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(self)")

		// Enforce HTTPS in production
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		return c.Next()
	}
}
