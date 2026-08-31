package iam

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for IAM administration.
type Handler struct {
	service *Service
}

// NewHandler creates a new IAM handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers IAM admin routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	for _, path := range []string{"", "/iam"} {
		iam := router.Group(path)

		// Roles
		iam.Get("/roles", h.ListRoles)
		iam.Post("/roles", h.CreateRole)
		iam.Put("/roles/:id/permissions", h.AssignRolePermissions)

		// Menus & Actions
		iam.Get("/menus", h.ListMenus)
		iam.Get("/menus/:menuId/actions", h.ListMenuActions)

		// User role assignment & district scoping
		iam.Post("/users/:userId/roles", h.AssignUserRole)
		iam.Get("/users/:userId/districts", h.GetUserDistrictScopes)
		iam.Post("/users/:userId/districts", h.AssignUserDistrictScopes)

		// Permission overrides & ABAC
		iam.Post("/overrides", h.CreateOverride)
		iam.Post("/abac-policies", h.CreateABACPolicy)

		// Audit log
		iam.Get("/audit-log", h.ListAuditLogs)
	}
}

// ─── Handlers ───────────────────────────────────

func (h *Handler) ListRoles(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	roles, err := h.service.ListRoles(c.Context(), tx, sess.ActiveTenantID)
	if err != nil {
		return response.InternalError(c, "Failed to list roles")
	}

	return response.Success(c, roles)
}

func (h *Handler) CreateRole(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return response.BadRequest(c, "Valid role name is required")
	}

	role, err := h.service.CreateRole(c.Context(), tx, sess.ActiveTenantID, req.Name, req.Description)
	if err != nil {
		return response.InternalError(c, "Failed to create role")
	}

	return response.Created(c, role)
}

func (h *Handler) AssignRolePermissions(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)
	roleID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		MenuActionIDs []int `json:"menu_action_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.AssignRolePermissions(c.Context(), tx, roleID, sess.ActiveTenantID, req.MenuActionIDs); err != nil {
		return response.InternalError(c, "Failed to assign permissions to role")
	}

	return response.Success(c, fiber.Map{"message": "Role permissions updated"})
}

func (h *Handler) ListMenus(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	menus, err := h.service.ListMenus(c.Context(), tx)
	if err != nil {
		return response.InternalError(c, "Failed to list menus")
	}
	return response.Success(c, menus)
}

func (h *Handler) ListMenuActions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	menuID, err := strconv.Atoi(c.Params("menuId"))
	if err != nil {
		return response.BadRequest(c, "Invalid menu ID")
	}

	actions, err := h.service.ListMenuActions(c.Context(), tx, menuID)
	if err != nil {
		return response.InternalError(c, "Failed to list menu actions")
	}
	return response.Success(c, actions)
}

func (h *Handler) AssignUserRole(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req struct {
		TenantID int `json:"tenant_id"`
		RoleID   int `json:"role_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.AssignUserRole(c.Context(), tx, userID, int64(req.TenantID), req.RoleID, sess.UserID); err != nil {
		return response.InternalError(c, "Failed to assign role")
	}

	return response.Success(c, fiber.Map{"message": "Role assigned successfully"})
}

func (h *Handler) CreateOverride(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var req struct {
		UserID       int64      `json:"user_id"`
		TenantID     int64      `json:"tenant_id"`
		MenuActionID int        `json:"menu_action_id"`
		Effect       string     `json:"effect"`
		Reason       string     `json:"reason"`
		ValidFrom    *time.Time `json:"valid_from"`
		ValidUntil   *time.Time `json:"valid_until"`
	}
	if err := c.BodyParser(&req); err != nil || (req.Effect != "GRANT" && req.Effect != "REVOKE") {
		return response.BadRequest(c, "Valid user, action, reason, and effect (GRANT/REVOKE) are required")
	}

	if err := h.service.CreateOverride(c.Context(), tx, req.UserID, req.TenantID, req.MenuActionID, req.Effect, req.Reason, req.ValidFrom, req.ValidUntil, sess.UserID); err != nil {
		return response.InternalError(c, "Failed to create override")
	}

	return response.Created(c, fiber.Map{"message": "Override created successfully"})
}

func (h *Handler) CreateABACPolicy(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var req struct {
		UserID    int64           `json:"user_id"`
		Attribute string          `json:"attribute"`
		Value     json.RawMessage `json:"value"`
	}
	if err := c.BodyParser(&req); err != nil || req.Attribute == "" {
		return response.BadRequest(c, "Attribute and JSON value are required")
	}

	if err := h.service.CreateABACPolicy(c.Context(), tx, req.UserID, sess.ActiveTenantID, req.Attribute, req.Value); err != nil {
		return response.InternalError(c, "Failed to create ABAC policy")
	}

	return response.Created(c, fiber.Map{"message": "ABAC policy created"})
}

func (h *Handler) ListAuditLogs(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 50)
	offset := (page - 1) * perPage

	logs, total, err := h.service.ListAuditLogs(c.Context(), tx, perPage, offset)
	if err != nil {
		return response.InternalError(c, "Failed to fetch audit logs")
	}

	return response.Paginated(c, logs, page, perPage, total)
}

func (h *Handler) GetUserDistrictScopes(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	scopes, err := h.service.GetUserDistrictScopes(c.Context(), tx, userID, sess.ActiveTenantID)
	if err != nil {
		return response.InternalError(c, "Failed to get district scopes")
	}

	return response.Success(c, scopes)
}

func (h *Handler) AssignUserDistrictScopes(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req struct {
		DistrictIDs []int `json:"district_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.AssignUserDistrictScopes(c.Context(), tx, userID, sess.ActiveTenantID, req.DistrictIDs); err != nil {
		return response.InternalError(c, "Failed to assign district scopes")
	}

	return response.Success(c, fiber.Map{"message": "District scopes assigned successfully"})
}
