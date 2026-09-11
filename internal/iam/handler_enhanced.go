package iam

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// EnhancedHandler exposes HTTP endpoints for the new IAM system
type EnhancedHandler struct {
	service *EnhancedService
}

// NewEnhancedHandler creates a new enhanced IAM handler
func NewEnhancedHandler(service *EnhancedService) *EnhancedHandler {
	return &EnhancedHandler{service: service}
}

// RegisterEnhancedRoutes registers the new IAM endpoints
func (h *EnhancedHandler) RegisterEnhancedRoutes(router fiber.Router) {
	iam := router.Group("/iam")

	// Permission management
	iam.Get("/permissions", h.ListPermissions)
	iam.Post("/permissions", h.CreatePermission)
	iam.Get("/permissions/:id", h.GetPermission)

	// Permission groups
	iam.Get("/permission-groups", h.ListPermissionGroups)

	// User direct permissions
	iam.Get("/users/:userId/permissions-new", h.GetUserPermissions)
	iam.Post("/users/:userId/permissions-new", h.GrantUserPermission)
	iam.Delete("/users/:userId/permissions-new/:permissionId", h.RevokeUserPermission)

	// Resource grants
	iam.Get("/users/:userId/resource-grants", h.GetUserResourceGrants)
	iam.Post("/resource-grants", h.CreateResourceGrant)
	iam.Delete("/resource-grants/:id", h.RevokeResourceGrant)

	// Enhanced policy management
	iam.Get("/abac-policies-new", h.ListEnhancedABACPolicies)
	iam.Post("/abac-policies-new", h.CreateEnhancedABACPolicy)
	iam.Put("/abac-policies-new/:id", h.UpdateEnhancedABACPolicy)
	iam.Delete("/abac-policies-new/:id", h.DeleteEnhancedABACPolicy)

	// Enhanced role methods
	iam.Get("/roles/:id/permissions-new", h.GetRolePermissionsNew)
	iam.Put("/roles/:id/permissions-new", h.AssignRolePermissionsNew)

	// Permission checking API
	iam.Post("/check-permission", h.CheckPermission)
	iam.Post("/check-resource-access", h.CheckResourceAccess)

	// Effective permissions (new system)
	iam.Get("/users/:userId/permissions-new/effective", h.GetUserEffectivePermissionsNew)
}

// ─── Permission Handlers ─────────────────────────────────

func (h *EnhancedHandler) ListPermissions(c *fiber.Ctx) error {
	permissions, err := h.service.ListPermissions(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list permissions: "+err.Error())
	}
	return response.Success(c, fiber.Map{"data": permissions})
}

func (h *EnhancedHandler) GetPermission(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid permission ID")
	}

	permission, err := h.service.GetPermission(c.Context(), id)
	if err != nil {
		return response.InternalError(c, "Failed to get permission: "+err.Error())
	}
	if permission == nil {
		return response.NotFound(c, "Permission not found")
	}

	return response.Success(c, fiber.Map{"data": permission})
}

func (h *EnhancedHandler) CreatePermission(c *fiber.Ctx) error {
	var req struct {
		Resource    string `json:"resource"`
		Action      string `json:"action"`
		Scope       string `json:"scope"`
		Description string `json:"description"`
		IsSystem    bool   `json:"is_system"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Resource == "" || req.Action == "" {
		return response.BadRequest(c, "Resource and action are required")
	}

	// Validate scope
	validScopes := map[string]bool{"all": true, "own": true, "department": true, "custom": true}
	if req.Scope != "" && !validScopes[req.Scope] {
		return response.BadRequest(c, "Invalid scope. Must be one of: all, own, department, custom")
	}

	if req.Scope == "" {
		req.Scope = "all"
	}

	// Sanitize input
	req.Resource = strings.TrimSpace(req.Resource)
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	req.Description = strings.TrimSpace(req.Description)

	permission, err := h.service.CreatePermission(c.Context(), req.Resource, req.Action, req.Scope, req.Description, req.IsSystem)
	if err != nil {
		return response.InternalError(c, "Failed to create permission: "+err.Error())
	}

	return response.Created(c, permission)
}

// ─── Permission Group Handlers ──────────────────────────

func (h *EnhancedHandler) ListPermissionGroups(c *fiber.Ctx) error {
	groups, err := h.service.ListPermissionGroups(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list permission groups: "+err.Error())
	}
	return response.Success(c, groups)
}

// ─── User Permission Handlers ─────────────────────────────

func (h *EnhancedHandler) GetUserPermissions(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	permissions, err := h.service.GetUserPermissions(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to get user permissions: "+err.Error())
	}

	return response.Success(c, permissions)
}

func (h *EnhancedHandler) GrantUserPermission(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req struct {
		PermissionID int        `json:"permission_id"`
		Effect       string     `json:"effect"`
		Reason       string     `json:"reason"`
		ValidFrom    *time.Time `json:"valid_from"`
		ValidUntil   *time.Time `json:"valid_until"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.PermissionID <= 0 {
		return response.BadRequest(c, "Valid permission_id is required")
	}

	if req.Effect == "" {
		req.Effect = "GRANT"
	}

	sess := middleware.SessionFromCtx(c)
	grantedBy := int64(1)
	if sess != nil {
		grantedBy = sess.UserID
	}

	err = h.service.GrantUserPermission(c.Context(), userID, req.PermissionID, req.Effect, req.Reason, req.ValidFrom, req.ValidUntil, grantedBy)
	if err != nil {
		return response.InternalError(c, "Failed to grant user permission: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "User permission granted successfully"})
}

func (h *EnhancedHandler) RevokeUserPermission(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	permissionID, err := strconv.Atoi(c.Params("permissionId"))
	if err != nil {
		return response.BadRequest(c, "Invalid permission ID")
	}

	err = h.service.RevokeUserPermission(c.Context(), userID, permissionID)
	if err != nil {
		return response.InternalError(c, "Failed to revoke user permission: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "User permission revoked successfully"})
}

// ─── Resource Grant Handlers ─────────────────────────────

func (h *EnhancedHandler) GetUserResourceGrants(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	grants, err := h.service.GetUserResourceGrants(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to get user resource grants: "+err.Error())
	}

	return response.Success(c, grants)
}

func (h *EnhancedHandler) CreateResourceGrant(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req struct {
		UserID       int64           `json:"user_id"`
		ResourceType string          `json:"resource_type"`
		ResourceID   string          `json:"resource_id"`
		PermissionID int             `json:"permission_id"`
		ExpiresAt    *time.Time      `json:"expires_at"`
		Conditions   json.RawMessage `json:"conditions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.UserID <= 0 || req.ResourceType == "" || req.ResourceID == "" || req.PermissionID <= 0 {
		return response.BadRequest(c, "user_id, resource_type, resource_id, and permission_id are required")
	}

	grant, err := h.service.CreateResourceGrant(c.Context(), req.UserID, req.ResourceType, req.ResourceID, req.PermissionID, sess.UserID, req.ExpiresAt, req.Conditions)
	if err != nil {
		return response.InternalError(c, "Failed to create resource grant: "+err.Error())
	}

	return response.Created(c, grant)
}

func (h *EnhancedHandler) RevokeResourceGrant(c *fiber.Ctx) error {
	grantID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid grant ID")
	}

	err = h.service.RevokeResourceGrant(c.Context(), grantID)
	if err != nil {
		return response.InternalError(c, "Failed to revoke resource grant: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Resource grant revoked successfully"})
}

// ─── Enhanced ABAC Policy Handlers ───────────────────────

func (h *EnhancedHandler) ListEnhancedABACPolicies(c *fiber.Ctx) error {
	policies, err := h.service.ListEnhancedABACPolicies(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list ABAC policies: "+err.Error())
	}
	return response.Success(c, policies)
}

func (h *EnhancedHandler) CreateEnhancedABACPolicy(c *fiber.Ctx) error {
	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		PolicyType  string          `json:"policy_type"`
		TargetID    *int64          `json:"target_id"`
		Effect      string          `json:"effect"`
		Priority    int             `json:"priority"`
		Conditions  json.RawMessage `json:"conditions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Name == "" || req.PolicyType == "" {
		return response.BadRequest(c, "Name and policy_type are required")
	}

	if req.Effect == "" {
		req.Effect = "allow"
	}

	if req.Priority == 0 {
		req.Priority = 0
	}

	policy, err := h.service.CreateEnhancedABACPolicy(c.Context(), req.Name, req.Description, req.PolicyType, req.TargetID, req.Effect, req.Priority, req.Conditions)
	if err != nil {
		return response.InternalError(c, "Failed to create ABAC policy: "+err.Error())
	}

	return response.Created(c, policy)
}

func (h *EnhancedHandler) UpdateEnhancedABACPolicy(c *fiber.Ctx) error {
	policyID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid policy ID")
	}

	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Effect      string          `json:"effect"`
		Priority    int             `json:"priority"`
		Conditions  json.RawMessage `json:"conditions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	policy, err := h.service.UpdateEnhancedABACPolicy(c.Context(), policyID, req.Name, req.Description, req.Effect, req.Priority, req.Conditions)
	if err != nil {
		return response.InternalError(c, "Failed to update ABAC policy: "+err.Error())
	}

	return response.Success(c, policy)
}

func (h *EnhancedHandler) DeleteEnhancedABACPolicy(c *fiber.Ctx) error {
	policyID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid policy ID")
	}

	err = h.service.DeleteEnhancedABACPolicy(c.Context(), policyID)
	if err != nil {
		return response.InternalError(c, "Failed to delete ABAC policy: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "ABAC policy deleted successfully"})
}

// ─── Enhanced Role Handlers ─────────────────────────────

func (h *EnhancedHandler) GetRolePermissionsNew(c *fiber.Ctx) error {
	roleID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	permissions, err := h.service.GetRolePermissionsNew(c.Context(), roleID)
	if err != nil {
		return response.InternalError(c, "Failed to get role permissions: "+err.Error())
	}

	return response.Success(c, permissions)
}

func (h *EnhancedHandler) AssignRolePermissionsNew(c *fiber.Ctx) error {
	roleID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid role ID")
	}

	var req struct {
		PermissionIDs []int `json:"permission_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate permission IDs
	if len(req.PermissionIDs) == 0 {
		return response.BadRequest(c, "At least one permission ID is required")
	}

	// Remove duplicates and validate
	uniqueIDs := make(map[int]bool)
	validIDs := []int{}
	for _, id := range req.PermissionIDs {
		if id <= 0 {
			return response.BadRequest(c, "Permission IDs must be positive integers")
		}
		if !uniqueIDs[id] {
			uniqueIDs[id] = true
			validIDs = append(validIDs, id)
		}
	}

	sess := middleware.SessionFromCtx(c)
	grantedBy := int64(1)
	if sess != nil {
		grantedBy = sess.UserID
	}

	err = h.service.AssignRolePermissionsNew(c.Context(), roleID, validIDs, grantedBy)
	if err != nil {
		return response.InternalError(c, "Failed to assign role permissions: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Role permissions assigned successfully"})
}

// ─── Permission Checking API ────────────────────────────

func (h *EnhancedHandler) CheckPermission(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req struct {
		Resource string `json:"resource"`
		Action   string `json:"action"`
		Scope    string `json:"scope"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Resource == "" || req.Action == "" {
		return response.BadRequest(c, "Resource and action are required")
	}

	if req.Scope == "" {
		req.Scope = "all"
	}

	allowed, err := h.service.Can(c.Context(), sess.UserID, req.Resource, req.Action, req.Scope)
	if err != nil {
		return response.InternalError(c, "Failed to check permission: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"allowed": allowed,
		"user_id": sess.UserID,
		"permission": fmt.Sprintf("%s:%s:%s", req.Resource, req.Action, req.Scope),
	})
}

func (h *EnhancedHandler) CheckResourceAccess(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	if sess == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	var req struct {
		Resource    string `json:"resource"`
		Action      string `json:"action"`
		ResourceID  string `json:"resource_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Resource == "" || req.Action == "" || req.ResourceID == "" {
		return response.BadRequest(c, "Resource, action, and resource_id are required")
	}

	allowed, err := h.service.CanAccessResource(c.Context(), sess.UserID, req.Resource, req.Action, req.ResourceID)
	if err != nil {
		return response.InternalError(c, "Failed to check resource access: "+err.Error())
	}

	return response.Success(c, fiber.Map{
		"allowed": allowed,
		"user_id": sess.UserID,
		"resource": req.Resource,
		"action": req.Action,
		"resource_id": req.ResourceID,
	})
}

// ─── Enhanced Effective Permissions ───────────────────────

func (h *EnhancedHandler) GetUserEffectivePermissionsNew(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	permissions, err := h.service.GetUserEffectivePermissionsNew(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to get user effective permissions: "+err.Error())
	}

	return response.Success(c, permissions)
}