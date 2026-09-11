package iam

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for IAM administration and permission evaluation.
type Handler struct {
	service       *Service
	enhancedService *EnhancedService
}

// NewHandler creates a new IAM handler.
func NewHandler(service *Service, enhancedService *EnhancedService) *Handler {
	return &Handler{
		service:       service,
		enhancedService: enhancedService,
	}
}

// RegisterRoutes registers IAM admin and governance routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	for _, path := range []string{"", "/iam"} {
		iam := router.Group(path)

		// 1. Roles & Permission Matrix (Legacy)
		iam.Get("/roles", h.ListRoles)
		iam.Post("/roles", h.CreateRole)
		iam.Get("/roles/:id", h.GetRole)
		iam.Put("/roles/:id", h.UpdateRole)
		iam.Delete("/roles/:id", h.DeleteRole)
		iam.Post("/roles/:id/clone", h.CloneRole)
		iam.Post("/roles/:id/template", h.ApplyRoleTemplate)
		iam.Get("/roles/:id/matrix", h.GetRolePermissionMatrix)
		iam.Put("/roles/:id/permissions", h.AssignRolePermissions)

		// 2. Menus & Actions (Legacy)
		iam.Get("/menus", h.ListMenus)
		iam.Get("/menus/:menuId/actions", h.ListMenuActions)

		// 3. Staff Governance & Scoping
		iam.Get("/staff", h.ListStaffWithRoles)

		// 4. User Role & Scoping
		iam.Post("/users/:userId/roles", h.AssignUserRole)
		iam.Get("/users/:userId/categories", h.GetUserCategoryScopes)
		iam.Post("/users/:userId/categories", h.AssignUserCategoryScopes)
		iam.Get("/users/:userId/districts", h.GetUserCategoryScopes)
		iam.Post("/users/:userId/districts", h.AssignUserCategoryScopes)

		// 5. Effective Permissions (Legacy)
		iam.Get("/users/:userId/permissions", h.GetUserEffectivePermissions)
		iam.Get("/me/permissions", h.GetMyEffectivePermissions)
		iam.Get("/me/menus", h.ListMyMenus)

		// 6. Overrides & ABAC Policies (Legacy)
		iam.Post("/overrides", h.CreateOverride)
		iam.Post("/abac-policies", h.CreateABACPolicy)

		// 7. Security & Permission Audit Log
		iam.Get("/audit-log", h.ListAuditLogs)

		// 8. Enhanced RBAC/ABAC APIs
		iam.Get("/permissions", h.ListAllPermissions)
		iam.Get("/roles/:id/permissions-new", h.GetRolePermissionsNew)
		iam.Post("/roles/:id/permissions", h.AssignRolePermissionsNew)
		iam.Post("/roles/:id/dynamic-permissions", h.AssignDynamicPermissions)
		iam.Get("/roles/:id/dynamic-permissions", h.GetDynamicPermissions)
		iam.Delete("/dynamic-permissions/:assignmentId", h.RevokeDynamicPermission)
		
		// 9. Approval Workflow APIs
		iam.Post("/approval-requests", h.CreateApprovalRequest)
		iam.Get("/approval-requests", h.ListApprovalRequests)
		iam.Post("/approval-requests/:id/approve", h.ApproveRequest)
		iam.Post("/approval-requests/:id/reject", h.RejectRequest)
		iam.Get("/approval-requests/pending", h.GetPendingApprovals)
		
		// 10. Approval Assignment APIs (SuperAdmin)
		iam.Post("/approval-assignments", h.AssignApprovalRights)
		iam.Get("/approval-assignments", h.GetApprovalAssignments)
		iam.Delete("/approval-assignments/:id", h.RevokeApprovalRights)
	}

	// Register enhanced routes if enhanced service is available
	if h.enhancedService != nil {
		enhancedHandler := NewEnhancedHandler(h.enhancedService)
		enhancedHandler.RegisterEnhancedRoutes(router)
	}
}

// ─── Handlers ───────────────────────────────────

func (h *Handler) ListRoles(c *fiber.Ctx) error {
	var roles []Role
	var err error
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		roles, err = h.service.ListRoles(c.Context(), tx)
	} else {
		roles, err = h.service.ListRolesDirect(c.Context())
	}

	if err != nil {
		return response.InternalError(c, "Failed to list roles: "+err.Error())
	}
	return response.Success(c, roles)
}

func (h *Handler) GetRole(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	role, err := h.service.GetRole(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to get role: "+err.Error())
	}
	if role == nil {
		return response.NotFound(c, "Role not found")
	}

	return response.Success(c, role)
}

func (h *Handler) CreateRole(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return response.BadRequest(c, "Role name is required")
	}

	role, err := h.service.CreateRole(c.Context(), tx, 1, strings.TrimSpace(req.Name), req.Description)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") || strings.Contains(err.Error(), "23505") {
			return response.BadRequest(c, fmt.Sprintf("A role named '%s' already exists. Please choose a different name.", req.Name))
		}
		return response.InternalError(c, "Failed to create role: "+err.Error())
	}
	return response.Created(c, role)
}

func (h *Handler) UpdateRole(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return response.BadRequest(c, "Role name is required")
	}

	role, err := h.service.UpdateRole(c.Context(), tx, id, strings.TrimSpace(req.Name), req.Description)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") || strings.Contains(err.Error(), "23505") {
			return response.BadRequest(c, fmt.Sprintf("A role named '%s' already exists. Please choose a different name.", req.Name))
		}
		return response.InternalError(c, "Failed to update role: "+err.Error())
	}
	return response.Success(c, role)
}

func (h *Handler) DeleteRole(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	if err := h.service.DeleteRole(c.Context(), tx, id); err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Role deleted successfully"})
}

func (h *Handler) CloneRole(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid source role ID")
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return response.BadRequest(c, "New role name is required")
	}

	newRole, err := h.service.CloneRole(c.Context(), tx, id, strings.TrimSpace(req.Name), req.Description)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") || strings.Contains(err.Error(), "23505") {
			return response.BadRequest(c, fmt.Sprintf("A role named '%s' already exists. Please choose a different name.", req.Name))
		}
		return response.InternalError(c, "Failed to clone role: "+err.Error())
	}
	return response.Created(c, newRole)
}

func (h *Handler) GetRolePermissionMatrix(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var matrix []MenuMatrixItem
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		matrix, err = h.service.GetRolePermissionMatrix(c.Context(), tx, id)
	} else {
		matrix, err = h.service.GetRolePermissionMatrixDirect(c.Context(), id)
	}

	if err != nil {
		return response.InternalError(c, "Failed to get permission matrix: "+err.Error())
	}
	return response.Success(c, matrix)
}

func (h *Handler) AssignRolePermissions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		ActionIDs []int `json:"action_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.AssignRolePermissions(c.Context(), tx, id, 1, req.ActionIDs); err != nil {
		return response.InternalError(c, "Failed to assign permissions: "+err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Permissions updated successfully"})
}

func (h *Handler) ListMenus(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	var menus []Menu
	var err error

	if sess != nil {
		if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
			menus, err = h.service.ListMenusForUser(c.Context(), tx, sess.UserID, sess.IsSuperAdmin)
		} else {
			menus, err = h.service.ListMenusForUser(c.Context(), nil, sess.UserID, sess.IsSuperAdmin)
		}
	} else if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		menus, err = h.service.ListMenus(c.Context(), tx)
	} else {
		menus, err = h.service.ListMenusDirect(c.Context())
	}

	if err != nil {
		return response.InternalError(c, "Failed to list menus: "+err.Error())
	}
	return response.Success(c, menus)
}

func (h *Handler) ListMyMenus(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Authentication required")
	}
	var menus []Menu
	var err error
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		menus, err = h.service.ListMenusForUser(c.Context(), tx, sess.UserID, sess.IsSuperAdmin)
	} else {
		menus, err = h.service.ListMenusForUser(c.Context(), nil, sess.UserID, sess.IsSuperAdmin)
	}
	if err != nil {
		return response.InternalError(c, "Failed to list menus: "+err.Error())
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
		return response.InternalError(c, "Failed to list menu actions: "+err.Error())
	}
	return response.Success(c, actions)
}

func (h *Handler) AssignUserRole(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req struct {
		RoleID int `json:"role_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.RoleID <= 0 {
		return response.BadRequest(c, "Valid role_id is required")
	}

	assignedBy := int64(1)
	if sess != nil {
		assignedBy = sess.UserID
	}

	if err := h.service.AssignUserRole(c.Context(), tx, userID, 1, req.RoleID, assignedBy); err != nil {
		return response.InternalError(c, "Failed to assign user role: "+err.Error())
	}
	return response.Success(c, fiber.Map{"message": "User role assigned successfully"})
}

func (h *Handler) GetUserCategoryScopes(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	scopes, err := h.service.GetUserCategoryScopes(c.Context(), tx, userID)
	if err != nil {
		return response.InternalError(c, "Failed to get category scopes: "+err.Error())
	}
	return response.Success(c, scopes)
}

func (h *Handler) AssignUserCategoryScopes(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req struct {
		CategoryIDs []int `json:"category_ids"`
		DistrictIDs []int `json:"district_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	catIDs := req.CategoryIDs
	if len(catIDs) == 0 && len(req.DistrictIDs) > 0 {
		catIDs = req.DistrictIDs
	}

	assignedBy := int64(1)
	if sess != nil {
		assignedBy = sess.UserID
	}

	if err := h.service.AssignUserCategoryScopes(c.Context(), tx, userID, catIDs, assignedBy); err != nil {
		return response.InternalError(c, "Failed to assign category scopes: "+err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Editorial bureau category scopes assigned successfully"})
}

func (h *Handler) GetUserEffectivePermissions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	perms, err := h.service.GetUserEffectivePermissions(c.Context(), tx, userID)
	if err != nil {
		return response.InternalError(c, "Failed to resolve effective permissions: "+err.Error())
	}
	return response.Success(c, perms)
}

func (h *Handler) GetMyEffectivePermissions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	perms, err := h.service.GetUserEffectivePermissions(c.Context(), tx, sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to resolve permissions: "+err.Error())
	}
	return response.Success(c, perms)
}

func (h *Handler) CreateOverride(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	var req struct {
		UserID       int64      `json:"user_id"`
		MenuActionID int        `json:"menu_action_id"`
		Effect       string     `json:"effect"`
		Reason       string     `json:"reason"`
		ValidFrom    *time.Time `json:"valid_from"`
		ValidUntil   *time.Time `json:"valid_until"`
	}
	if err := c.BodyParser(&req); err != nil || req.UserID <= 0 || req.MenuActionID <= 0 {
		return response.BadRequest(c, "user_id and menu_action_id are required")
	}

	grantedBy := int64(1)
	if sess != nil {
		grantedBy = sess.UserID
	}

	if err := h.service.CreateOverride(c.Context(), tx, req.UserID, 1, req.MenuActionID, req.Effect, req.Reason, req.ValidFrom, req.ValidUntil, grantedBy); err != nil {
		return response.InternalError(c, "Failed to create override: "+err.Error())
	}
	return response.Created(c, fiber.Map{"message": "Override created successfully"})
}

func (h *Handler) CreateABACPolicy(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	var req struct {
		UserID    int64           `json:"user_id"`
		Attribute string          `json:"attribute"`
		Value     json.RawMessage `json:"value"`
	}
	if err := c.BodyParser(&req); err != nil || req.UserID <= 0 || req.Attribute == "" {
		return response.BadRequest(c, "user_id and attribute are required")
	}

	if err := h.service.CreateABACPolicy(c.Context(), tx, req.UserID, 1, req.Attribute, req.Value); err != nil {
		return response.InternalError(c, "Failed to create ABAC policy: "+err.Error())
	}
	return response.Created(c, fiber.Map{"message": "ABAC policy created successfully"})
}

func (h *Handler) ListAuditLogs(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	logs, total, err := h.service.ListAuditLogs(c.Context(), tx, limit, offset)
	if err != nil {
		return response.InternalError(c, "Failed to list audit logs: "+err.Error())
	}

	return response.Paginated(c, logs, (offset/limit)+1, limit, total)
}

// ─── Enhanced RBAC/ABAC Handlers ───────────────────────────

func (h *Handler) ListAllPermissions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	
	permissions, err := h.service.ListAllPermissions(c.Context(), tx)
	if err != nil {
		return response.InternalError(c, "Failed to list permissions: "+err.Error())
	}
	
	return response.Success(c, permissions)
}

func (h *Handler) GetRolePermissionsNew(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	permissions, err := h.service.GetRolePermissions(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to get role permissions: "+err.Error())
	}
	
	return response.Success(c, permissions)
}

func (h *Handler) AssignRolePermissionsNew(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		Grant []int  `json:"grant"`
		Revoke []int `json:"revoke"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if len(req.Grant) > 0 {
		if err := h.service.AssignRolePermissionsBulk(c.Context(), tx, id, req.Grant, sess.UserID); err != nil {
			return response.InternalError(c, "Failed to grant permissions: "+err.Error())
		}
	}

	if len(req.Revoke) > 0 {
		if err := h.service.RevokeRolePermissionsBulk(c.Context(), tx, id, req.Revoke, sess.UserID); err != nil {
			return response.InternalError(c, "Failed to revoke permissions: "+err.Error())
		}
	}

	return response.Success(c, fiber.Map{"message": "Permissions updated successfully"})
}

func (h *Handler) AssignDynamicPermissions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		PermissionIDs []int    `json:"permission_ids"`
		ExpiresAt     *time.Time `json:"expires_at"`
		Reason        string    `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.AssignDynamicPermissions(c.Context(), tx, id, req.PermissionIDs, sess.UserID, req.ExpiresAt, req.Reason); err != nil {
		return response.InternalError(c, "Failed to assign dynamic permissions: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Dynamic permissions assigned successfully"})
}

func (h *Handler) GetDynamicPermissions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	permissions, err := h.service.GetDynamicPermissions(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to get dynamic permissions: "+err.Error())
	}

	return response.Success(c, permissions)
}

func (h *Handler) RevokeDynamicPermission(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	assignmentId, err := strconv.Atoi(c.Params("assignmentId"))
	if err != nil {
		return response.BadRequest(c, "Invalid assignment ID")
	}

	if err := h.service.RevokeDynamicPermission(c.Context(), tx, assignmentId, sess.UserID); err != nil {
		return response.InternalError(c, "Failed to revoke dynamic permission: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Dynamic permission revoked successfully"})
}

// ─── Approval Workflow Handlers ───────────────────────────

func (h *Handler) CreateApprovalRequest(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	var req struct {
		ResourceType string `json:"resource_type"`
		ResourceID   int64  `json:"resource_id"`
		Comments     string `json:"comments"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	requestID, err := h.service.CreateApprovalRequest(c.Context(), tx, req.ResourceType, req.ResourceID, sess.UserID, req.Comments)
	if err != nil {
		return response.InternalError(c, "Failed to create approval request: "+err.Error())
	}

	return response.Success(c, fiber.Map{"request_id": requestID, "status": "pending"})
}

func (h *Handler) ListApprovalRequests(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	requests, err := h.service.ListApprovalRequests(c.Context(), tx, sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to list approval requests: "+err.Error())
	}

	return response.Success(c, requests)
}

func (h *Handler) GetPendingApprovals(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	requests, err := h.service.GetPendingApprovals(c.Context(), tx, sess.UserID)
	if err != nil {
		return response.InternalError(c, "Failed to get pending approvals: "+err.Error())
	}

	return response.Success(c, requests)
}

func (h *Handler) ApproveRequest(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid request ID")
	}

	var req struct {
		Comments string `json:"comments"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.ApproveRequest(c.Context(), tx, id, sess.UserID, req.Comments); err != nil {
		return response.InternalError(c, "Failed to approve request: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Request approved successfully"})
}

func (h *Handler) RejectRequest(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid request ID")
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.RejectRequest(c.Context(), tx, id, sess.UserID, req.Reason); err != nil {
		return response.InternalError(c, "Failed to reject request: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Request rejected successfully"})
}

// ─── Approval Assignment Handlers (SuperAdmin) ───────────────────────────

func (h *Handler) AssignApprovalRights(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	var req struct {
		UserID       int64       `json:"user_id"`
		ResourceType string      `json:"resource_type"`
		CanApprove   bool        `json:"can_approve"`
		ExpiresAt    *time.Time  `json:"expires_at"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.AssignApprovalRights(c.Context(), tx, req.UserID, req.ResourceType, req.CanApprove, sess.UserID, req.ExpiresAt); err != nil {
		return response.InternalError(c, "Failed to assign approval rights: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Approval rights assigned successfully"})
}

func (h *Handler) GetApprovalAssignments(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	assignments, err := h.service.GetApprovalAssignments(c.Context(), tx)
	if err != nil {
		return response.InternalError(c, "Failed to get approval assignments: "+err.Error())
	}

	return response.Success(c, assignments)
}

func (h *Handler) RevokeApprovalRights(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid assignment ID")
	}

	if err := h.service.RevokeApprovalRights(c.Context(), tx, id, sess.UserID); err != nil {
		return response.InternalError(c, "Failed to revoke approval rights: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Approval rights revoked successfully"})
}

func (h *Handler) ApplyRoleTemplate(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		Template string `json:"template"`
	}
	if err := c.BodyParser(&req); err != nil || req.Template == "" {
		return response.BadRequest(c, "Template preset name is required (e.g. editor, reporter, moderator, fact_checker)")
	}

	if err := h.service.ApplyRoleTemplate(c.Context(), tx, id, req.Template); err != nil {
		return response.InternalError(c, "Failed to apply template: "+err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Role template applied successfully"})
}

func (h *Handler) ListStaffWithRoles(c *fiber.Ctx) error {
	search := c.Query("search", "")
	var list []StaffUserRoleSummary
	var err error

	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		list, err = h.service.ListStaffWithRoles(c.Context(), tx, search)
	} else {
		list, err = h.service.ListStaffWithRolesDirect(c.Context(), search)
	}

	if err != nil {
		return response.InternalError(c, "Failed to list staff: "+err.Error())
	}
	return response.Success(c, list)
}
