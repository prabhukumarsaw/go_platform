package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// EdgeCache sets standard CDN and browser caching headers.
// s-maxage directs Cloudflare/CloudFront edge PoPs, while stale-while-revalidate allows instant serving while fetching updates in background.
func EdgeCache(sMaxAgeSec int, swrSec int) fiber.Handler {
	cacheControlVal := fmt.Sprintf("public, max-age=0, s-maxage=%d, stale-while-revalidate=%d", sMaxAgeSec, swrSec)

	return func(c *fiber.Ctx) error {
		// Only cache GET and HEAD requests
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead {
			c.Set("Cache-Control", cacheControlVal)
			c.Set("CDN-Cache-Control", fmt.Sprintf("max-age=%d", sMaxAgeSec))
		}
		return c.Next()
	}
}
