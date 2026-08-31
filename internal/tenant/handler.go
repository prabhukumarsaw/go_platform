package tenant

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for tenant management.
type Handler struct {
	service *Service
}

// NewHandler creates a new tenant handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers tenant admin routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	t := router.Group("/tenants")
	t.Get("/", h.List)
	t.Post("/", h.Create)
	t.Get("/:id/districts", h.ListDistricts)
	t.Post("/:id/districts", h.CreateDistrict)
}

// List returns all tenants.
func (h *Handler) List(c *fiber.Ctx) error {
	tenants, err := h.service.ListTenants(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list tenants")
	}
	return response.Success(c, tenants)
}

// CreateTenantRequest is the request body.
type CreateTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Create creates a new tenant.
func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	if req.Name == "" || req.Slug == "" {
		return response.BadRequest(c, "Name and slug are required")
	}

	tenant, err := h.service.CreateTenant(c.Context(), req.Name, req.Slug)
	if err != nil {
		return response.InternalError(c, "Failed to create tenant")
	}
	return response.Created(c, tenant)
}

// ListDistricts returns districts for a tenant.
func (h *Handler) ListDistricts(c *fiber.Ctx) error {
	tenantID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid tenant ID")
	}

	districts, err := h.service.ListDistricts(c.Context(), tenantID)
	if err != nil {
		return response.InternalError(c, "Failed to list districts")
	}
	return response.Success(c, districts)
}

// CreateDistrictRequest is the request body.
type CreateDistrictRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateDistrict creates a new district.
func (h *Handler) CreateDistrict(c *fiber.Ctx) error {
	tenantID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid tenant ID")
	}

	var req CreateDistrictRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	if req.Name == "" || req.Slug == "" {
		return response.BadRequest(c, "Name and slug are required")
	}

	district, err := h.service.CreateDistrict(c.Context(), tenantID, req.Name, req.Slug)
	if err != nil {
		return response.InternalError(c, "Failed to create district")
	}
	return response.Created(c, district)
}
