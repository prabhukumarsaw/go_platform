package employee

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	emp := router.Group("/employees")
	emp.Get("/", h.List)
	emp.Get("/:id", h.GetByID)
	emp.Post("/", h.Onboard)
	emp.Patch("/:id/status", h.UpdateStatus)
	emp.Delete("/:id", h.Delete)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	department := c.Query("department")
	search := c.Query("q")
	if search == "" {
		search = c.Query("search")
	}

	employees, err := h.service.ListEmployees(c.Context(), tx, department, search)
	if err != nil {
		return response.InternalError(c, "Failed to list employees: "+err.Error())
	}

	return response.Success(c, employees)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid employee ID")
	}

	emp, err := h.service.GetEmployeeByID(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to get employee: "+err.Error())
	}
	if emp == nil {
		return response.NotFound(c, "Employee not found")
	}

	return response.Success(c, emp)
}

func (h *Handler) Onboard(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	var input OnboardEmployeeInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	emp, err := h.service.OnboardEmployee(c.Context(), tx, input)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Created(c, emp)
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid employee ID")
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.UpdateStatus(c.Context(), tx, id, req.IsActive); err != nil {
		return response.InternalError(c, "Failed to update employee status: "+err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Employee status updated"})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid employee ID")
	}

	if err := h.service.DeleteEmployee(c.Context(), tx, id); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Employee removed successfully"})
}
