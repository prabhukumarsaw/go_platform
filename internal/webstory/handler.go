package webstory

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	stories := router.Group("/web-stories")
	stories.Get("/", h.List)
	stories.Get("/:slug", h.GetBySlug)
}

func (h *Handler) RegisterStudioRoutes(router fiber.Router) {
	router.Post("/studio/web-stories", h.Create)
}

func (h *Handler) List(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	language := c.Query("language", "hi")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	offset := (page - 1) * limit

	stories, err := h.service.ListWebStories(c.Context(), tx, language, limit, offset)
	if err != nil {
		return response.InternalError(c, "Failed to fetch web stories: "+err.Error())
	}

	return response.Success(c, stories)
}

func (h *Handler) GetBySlug(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	slug := c.Params("slug")
	language := c.Query("language", "hi")

	story, err := h.service.GetWebStoryBySlug(c.Context(), tx, slug, language)
	if err != nil {
		return response.InternalError(c, "Failed to get web story: "+err.Error())
	}
	if story == nil {
		return response.NotFound(c, "Web story not found")
	}

	return response.Success(c, story)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var req struct {
		Title      string          `json:"title"`
		Language   string          `json:"language"`
		CoverImage string          `json:"cover_image"`
		Slides     json.RawMessage `json:"slides"`
	}
	if err := c.BodyParser(&req); err != nil || req.Title == "" || req.CoverImage == "" {
		return response.BadRequest(c, "Title and Cover image are required")
	}

	story, err := h.service.CreateWebStory(c.Context(), tx, int(sess.ActiveTenantID), sess.UserID, req.Title, req.Language, req.CoverImage, req.Slides)
	if err != nil {
		return response.InternalError(c, "Failed to create web story: "+err.Error())
	}

	return response.Created(c, story)
}
