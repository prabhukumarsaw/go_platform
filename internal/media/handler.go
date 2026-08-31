package media

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for media management.
type Handler struct {
	service *Service
}

// NewHandler creates a new media handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers media routes (staff only).
func (h *Handler) RegisterRoutes(router fiber.Router) {
	for _, path := range []string{"/media", "/studio/media"} {
		m := router.Group(path)
		m.Post("/upload", h.Upload)
		m.Get("/", h.List)
		m.Patch("/:id", h.UpdateMetadata)
		m.Delete("/:id", h.Delete)
	}
}

// Upload handles multipart file upload to local storage with category and folder tags.
func (h *Handler) Upload(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "No file provided")
	}

	category := c.FormValue("category", "news")
	folder := c.FormValue("folder", "general")

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return response.InternalError(c, "Failed to read uploaded file")
	}
	defer src.Close()

	// Detect MIME type from the multipart header
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	media, err := h.service.UploadFile(
		c.Context(), tx,
		int(sess.ActiveTenantID), sess.UserID,
		file.Filename, mimeType, category, folder, file.Size, src,
	)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Created(c, media)
}

// List returns media files with filtering by category, mimeType, and search.
func (h *Handler) List(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	tenantID := 1
	if sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	}

	category := c.Query("category")
	mimeType := c.Query("mime_type")
	search := c.Query("search")
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 20)

	items, total, err := h.service.ListMedia(c.Context(), tx, tenantID, category, mimeType, search, page, perPage)
	if err != nil {
		return response.InternalError(c, "Failed to list media: "+err.Error())
	}

	return response.Paginated(c, items, page, perPage, total)
}

// UpdateMetadata updates alt text, caption, category, and folder for a media item.
func (h *Handler) UpdateMetadata(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid media ID")
	}

	var body struct {
		AltText  string `json:"alt_text"`
		Caption  string `json:"caption"`
		Category string `json:"category"`
		Folder   string `json:"folder"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.UpdateMediaMetadata(c.Context(), tx, id, body.AltText, body.Caption, body.Category, body.Folder); err != nil {
		return response.InternalError(c, "Failed to update media metadata")
	}

	return response.Success(c, fiber.Map{"message": "Media metadata updated"})
}

// Delete removes a media file.
func (h *Handler) Delete(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid media ID")
	}

	if err := h.service.DeleteMedia(c.Context(), tx, id); err != nil {
		return response.InternalError(c, "Failed to delete media")
	}

	return response.Success(c, fiber.Map{"message": "Media deleted"})
}
