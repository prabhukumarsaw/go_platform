package middleware

import (
	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// IAMChecker is the interface that the IAM module must implement.
// Decoupled from the concrete IAMService to avoid circular imports.
type IAMChecker interface {
	Can(ctx interface{}, userID int64, action string) (bool, error)
}

// RequirePermission is a Fiber middleware that checks whether the authenticated
// user has the given permission (menu_action).
func RequirePermission(iam IAMChecker, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Not authenticated")
		}

		allowed, err := iam.Can(c.Context(), sess.UserID, action)
		if err != nil {
			return response.InternalError(c, "Permission check failed")
		}
		if !allowed {
			return response.Forbidden(c, "You do not have permission to perform this action")
		}

		return c.Next()
	}
}
