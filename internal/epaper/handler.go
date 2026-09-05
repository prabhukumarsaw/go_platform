package epaper

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

func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Get("/epapers", h.List)
}

func (h *Handler) RegisterStudioRoutes(router fiber.Router) {
	router.Post("/studio/epapers", h.Create)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	var districtID *int
	if dStr := c.Query("district_id"); dStr != "" {
		if id, err := strconv.Atoi(dStr); err == nil {
			districtID = &id
		}
	}

	dateStr := c.Query("date")

	epapers, err := h.service.ListEPapers(c.Context(), tx, districtID, dateStr)
	if err != nil {
		return response.InternalError(c, "Failed to fetch epapers: "+err.Error())
	}

	return response.Success(c, epapers)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	var req struct {
		DistrictID   *int   `json:"district_id"`
		EditionDate  string `json:"edition_date"`
		Title        string `json:"title"`
		PDFURL       string `json:"pdf_url"`
		ThumbnailURL string `json:"thumbnail_url"`
		PageCount    int    `json:"page_count"`
	}
	if err := c.BodyParser(&req); err != nil || req.EditionDate == "" || req.Title == "" || req.PDFURL == "" {
		return response.BadRequest(c, "Edition date, title, and PDF URL are required")
	}

	epaper, err := h.service.CreateEPaper(c.Context(), tx, req.DistrictID, req.EditionDate, req.Title, req.PDFURL, req.ThumbnailURL, req.PageCount)
	if err != nil {
		return response.InternalError(c, "Failed to create epaper: "+err.Error())
	}

	return response.Created(c, epaper)
}
