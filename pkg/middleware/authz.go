package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// PermissionEval is the request payload for a single RBAC/ABAC check.
type PermissionEval struct {
	UserID       int64
	IsSuperAdmin bool
	Action       string
	IPAddress    string
	UserAgent    string
	RequestID    string
	DeviceType   string
}

// PermissionEngine evaluates a canonical menu.ACTION string.
type PermissionEngine func(ctx context.Context, req PermissionEval) (bool, error)

// MenuPrefix maps a CMS menu to one or more API path prefixes (after /api/v1).
type MenuPrefix struct {
	Name      string
	APIPrefix string
}

// RequirePermission checks a specific permission atom (e.g. articles.PUBLISH).
func RequirePermission(engine PermissionEngine, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Not authenticated")
		}
		if sess.IsSuperAdmin {
			return c.Next()
		}

		allowed, err := engine(c.Context(), PermissionEval{
			UserID:       sess.UserID,
			IsSuperAdmin: sess.IsSuperAdmin,
			Action:       action,
			IPAddress:    c.IP(),
			UserAgent:    c.Get("User-Agent"),
			RequestID:    c.Get("X-Request-ID"),
			DeviceType:   DetectDeviceType(c.Get("User-Agent")),
		})
		if err != nil {
			return response.InternalError(c, "Permission check failed")
		}
		if !allowed {
			return response.Forbidden(c, "You do not have permission to perform this action")
		}
		return c.Next()
	}
}

// RequireMenuAccess maps the request path onto seeded menu api_prefix values
// and requires the HTTP-method action (VIEW/ADD/EDIT/DELETE/PUBLISH/APPROVE).
func RequireMenuAccess(engine PermissionEngine, menus []MenuPrefix) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Not authenticated")
		}
		if sess.IsSuperAdmin {
			return c.Next()
		}

		path := c.Path()
		if isRBACExempt(path) {
			return c.Next()
		}

		menuName := matchMenu(path, menus)
		if menuName == "" {
			return response.Forbidden(c, "You do not have permission to access this resource")
		}

		action := httpMethodToAction(c)
		allowed, err := engine(c.Context(), PermissionEval{
			UserID:       sess.UserID,
			IsSuperAdmin: sess.IsSuperAdmin,
			Action:       menuName + "." + action,
			IPAddress:    c.IP(),
			UserAgent:    c.Get("User-Agent"),
			RequestID:    c.Get("X-Request-ID"),
			DeviceType:   DetectDeviceType(c.Get("User-Agent")),
		})
		if err != nil {
			return response.InternalError(c, "Permission check failed")
		}
		if !allowed {
			return response.Forbidden(c, "You do not have permission to perform this action")
		}
		return c.Next()
	}
}

func isRBACExempt(path string) bool {
	exempt := []string{
		"/api/v1/iam/me",
		"/api/v1/auth",
		"/health",
	}
	for _, p := range exempt {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func matchMenu(path string, menus []MenuPrefix) string {
	best := ""
	bestLen := 0
	for _, m := range menus {
		for _, raw := range strings.Split(m.APIPrefix, ",") {
			prefix := strings.TrimSpace(raw)
			if prefix == "" {
				continue
			}
			full := prefix
			if !strings.HasPrefix(prefix, "/api/") {
				full = "/api/v1" + prefix
			}
			if strings.HasPrefix(path, full) && len(full) > bestLen {
				best = m.Name
				bestLen = len(full)
			}
		}
	}
	return best
}

func httpMethodToAction(c *fiber.Ctx) string {
	p := strings.ToLower(c.Path())
	if strings.Contains(p, "/transition") || strings.Contains(p, "/schedule") || strings.Contains(p, "/publish") {
		return "PUBLISH"
	}
	if strings.Contains(p, "/approve") {
		return "APPROVE"
	}
	switch c.Method() {
	case fiber.MethodPost:
		return "ADD"
	case fiber.MethodPut, fiber.MethodPatch:
		return "EDIT"
	case fiber.MethodDelete:
		return "DELETE"
	default:
		return "VIEW"
	}
}

// DetectDeviceType classifies User-Agent into mobile, tablet, or desktop.
func DetectDeviceType(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "tablet") || strings.Contains(u, "ipad"):
		return "tablet"
	case strings.Contains(u, "mobile") || strings.Contains(u, "android") || strings.Contains(u, "iphone"):
		return "mobile"
	default:
		return "desktop"
	}
}
