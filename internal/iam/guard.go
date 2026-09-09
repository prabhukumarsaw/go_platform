package iam

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// StaffRBAC enforces menu-level permissions using api_prefix values from the menus table.
func (h *Handler) StaffRBAC() fiber.Handler {
	engine := func(ctx context.Context, req middleware.PermissionEval) (bool, error) {
		return h.service.Evaluate(ctx, req.UserID, req.IsSuperAdmin, req.Action, req.IPAddress, req.UserAgent, req.RequestID, req.DeviceType)
	}

	return func(c *fiber.Ctx) error {
		menus, err := h.service.ListMenusDirect(c.Context())
		if err != nil {
			return response.Forbidden(c, "Access control is unavailable")
		}
		prefixes := make([]middleware.MenuPrefix, 0, len(menus))
		for _, m := range menus {
			if m.APIPrefix == "" {
				continue
			}
			prefixes = append(prefixes, middleware.MenuPrefix{Name: m.Name, APIPrefix: m.APIPrefix})
		}
		return middleware.RequireMenuAccess(engine, prefixes)(c)
	}
}
