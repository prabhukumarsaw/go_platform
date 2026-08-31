package middleware

import (
	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// IAMChecker is the interface that the IAM module must implement.
// Decoupled from the concrete IAMService to avoid circular imports.
type IAMChecker interface {
	Can(ctx interface{}, userID, tenantID int64, action string) (bool, error)
}

// RequirePermission is a Fiber middleware that checks whether the authenticated
// user has the given permission (menu_action) for their active tenant.
//
// Evaluation order (from §5 of the architecture):
//  1. super_admin bypass → ALLOW
//  2. user_permission override (REVOKE, not expired) → DENY
//  3. user_permission override (GRANT, not expired) + ABAC check → ALLOW/DENY
//  4. role → role_menu_action grant + ABAC check → ALLOW/DENY
//  5. no matching rule → DENY (default-deny)
//
// Must be placed after RequireAuth and RequireStaff in the middleware chain.
func RequirePermission(iam IAMChecker, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess := SessionFromCtx(c)
		if sess == nil {
			return response.Unauthorized(c, "Not authenticated")
		}

		allowed, err := iam.Can(c.Context(), sess.UserID, sess.ActiveTenantID, action)
		if err != nil {
			return response.InternalError(c, "Permission check failed")
		}
		if !allowed {
			return response.Forbidden(c, "You do not have permission to perform this action")
		}

		return c.Next()
	}
}
