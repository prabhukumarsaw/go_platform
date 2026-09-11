package iam

import (
	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// MigrationHandler exposes HTTP endpoints for IAM migration
type MigrationHandler struct {
	service *MigrationService
}

// NewMigrationHandler creates a new migration handler
func NewMigrationHandler(service *MigrationService) *MigrationHandler {
	return &MigrationHandler{service: service}
}

// RegisterMigrationRoutes registers migration endpoints
func (h *MigrationHandler) RegisterMigrationRoutes(router fiber.Router) {
	migration := router.Group("/iam/migration")

	migration.Post("/menu-actions", h.MigrateMenuActions)
	migration.Post("/role-permissions", h.MigrateRolePermissions)
	migration.Post("/user-overrides", h.MigrateUserOverrides)
	migration.Post("/full", h.RunFullMigration)
	migration.Get("/validate", h.ValidateMigration)
}

// MigrateMenuActions migrates menu actions to permissions
func (h *MigrationHandler) MigrateMenuActions(c *fiber.Ctx) error {
	result, err := h.service.MigrateMenuActionsToPermissions(c.Context())
	if err != nil {
		return response.InternalError(c, "Migration failed: "+err.Error())
	}
	return response.Success(c, result)
}

// MigrateRolePermissions migrates role permissions
func (h *MigrationHandler) MigrateRolePermissions(c *fiber.Ctx) error {
	result, err := h.service.MigrateRolePermissions(c.Context())
	if err != nil {
		return response.InternalError(c, "Migration failed: "+err.Error())
	}
	return response.Success(c, result)
}

// MigrateUserOverrides migrates user overrides
func (h *MigrationHandler) MigrateUserOverrides(c *fiber.Ctx) error {
	result, err := h.service.MigrateUserOverrides(c.Context())
	if err != nil {
		return response.InternalError(c, "Migration failed: "+err.Error())
	}
	return response.Success(c, result)
}

// RunFullMigration runs the complete migration
func (h *MigrationHandler) RunFullMigration(c *fiber.Ctx) error {
	result, err := h.service.RunFullMigration(c.Context())
	if err != nil {
		return response.InternalError(c, "Migration failed: "+err.Error())
	}
	return response.Success(c, result)
}

// ValidateMigration validates the migration
func (h *MigrationHandler) ValidateMigration(c *fiber.Ctx) error {
	result, err := h.service.ValidateMigration(c.Context())
	if err != nil {
		return response.InternalError(c, "Validation failed: "+err.Error())
	}
	return response.Success(c, result)
}