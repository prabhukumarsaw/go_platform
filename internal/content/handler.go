package content

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/middleware"
	"newsplatform/api/pkg/response"
)

// Handler exposes HTTP endpoints for articles and content.
type Handler struct {
	service *Service
}

// NewHandler creates a new content handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterPublicRoutes registers reader routes.
func (h *Handler) RegisterPublicRoutes(router fiber.Router) {
	// Public routes (reader)
	public := router.Group("/articles")
	public.Get("/", h.ListArticles)
	public.Get("/breaking", h.ListBreakingNews)
	public.Get("/featured", h.ListFeaturedNews)
	public.Get("/trending", h.ListTrendingNews)
	public.Get("/:slug", h.GetArticleBySlug)

	// Public taxonomy, search, live blogs, feed & home aggregator
	router.Get("/categories", h.ListCategories)
	router.Get("/search", h.Search)
	router.Get("/home", h.GetHomeFeed)
	router.Get("/feed/home", h.GetHomeFeed)
	router.Get("/live-blogs/:articleId/entries", h.ListLiveBlogEntries)
	router.Get("/stories/:storyId/variants", h.GetStoryVariants)
	router.Get("/feed", h.GetPersonalizedFeed)
}

// RegisterStudioRoutes registers staff studio routes.
func (h *Handler) RegisterStudioRoutes(router fiber.Router) {
	studio := router.Group("/studio/articles")
	studio.Get("/", h.StudioListArticles)
	studio.Post("/", h.CreateArticle)
	studio.Get("/:id", h.GetArticle)
	studio.Post("/:id/transition", h.TransitionStatus)
	studio.Get("/:id/versions", h.GetArticleVersions)

	// Studio Stories & Live blogs
	router.Post("/studio/stories", h.CreateStory)
	router.Post("/studio/live-blogs/:id/entries", h.AddLiveBlogEntry)
}

// ─── Public Handlers ────────────────────────────

// ListArticles returns published articles with full filtering capability.
func (h *Handler) ListArticles(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	tenantID := 1
	if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	} else if sess := middleware.SessionFromCtx(c); sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	}

	filter := ListArticlesFilter{
		TenantID:     tenantID,
		Status:       "published",
		Language:     c.Query("language", "en"),
		Category:     c.Query("category"),
		DistrictSlug: c.Query("district"),
		Period:       c.Query("period"),
		SortBy:       c.Query("sort", "latest"),
		Page:         c.QueryInt("page", 1),
		PerPage:      c.QueryInt("per_page", 20),
	}

	if authorID := c.Query("author_id"); authorID != "" {
		if id, err := strconv.ParseInt(authorID, 10, 64); err == nil {
			filter.AuthorID = &id
		}
	}

	if isBreaking := c.Query("is_breaking"); isBreaking != "" {
		b := isBreaking == "true" || isBreaking == "1"
		filter.IsBreaking = &b
	}

	if isFeatured := c.Query("is_featured"); isFeatured != "" {
		b := isFeatured == "true" || isFeatured == "1"
		filter.IsFeatured = &b
	}

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filter.DateFrom = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			filter.DateTo = &endOfDay
		}
	}

	articles, total, err := h.service.ListArticles(c.Context(), tx, filter)
	if err != nil {
		return response.InternalError(c, "Failed to list articles: "+err.Error())
	}

	return response.Paginated(c, articles, filter.Page, filter.PerPage, total)
}

// ListBreakingNews returns breaking news alerts.
func (h *Handler) ListBreakingNews(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	isBreaking := true

	filter := ListArticlesFilter{
		Status:     "published",
		Language:   c.Query("language", "en"),
		IsBreaking: &isBreaking,
		Page:       1,
		PerPage:    c.QueryInt("limit", 10),
	}

	articles, _, err := h.service.ListArticles(c.Context(), tx, filter)
	if err != nil {
		return response.InternalError(c, "Failed to fetch breaking news")
	}

	return response.Success(c, articles)
}

// ListFeaturedNews returns top editorial spotlight articles.
func (h *Handler) ListFeaturedNews(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	isFeatured := true

	filter := ListArticlesFilter{
		Status:     "published",
		Language:   c.Query("language", "en"),
		IsFeatured: &isFeatured,
		Page:       1,
		PerPage:    c.QueryInt("limit", 6),
	}

	articles, _, err := h.service.ListArticles(c.Context(), tx, filter)
	if err != nil {
		return response.InternalError(c, "Failed to fetch featured news")
	}

	return response.Success(c, articles)
}

// ListTrendingNews returns top-viewed trending stories.
func (h *Handler) ListTrendingNews(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	filter := ListArticlesFilter{
		Status:   "published",
		Language: c.Query("language", "en"),
		SortBy:   "trending",
		Page:     1,
		PerPage:  c.QueryInt("limit", 10),
	}

	articles, _, err := h.service.ListArticles(c.Context(), tx, filter)
	if err != nil {
		return response.InternalError(c, "Failed to fetch trending news")
	}

	return response.Success(c, articles)
}

// GetArticleBySlug retrieves a single article by slug.
func (h *Handler) GetArticleBySlug(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	slug := c.Params("slug")
	language := c.Query("language", "en")

	article, err := h.service.GetArticleBySlug(c.Context(), tx, slug, language)
	if err != nil {
		return response.InternalError(c, "Failed to get article: "+err.Error())
	}
	if article == nil {
		return response.NotFound(c, "Article not found")
	}

	return response.Success(c, article)
}

// ListCategories returns available categories for the active tenant.
func (h *Handler) ListCategories(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	tenantID := 1
	if sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	}

	categories, err := h.service.ListCategories(c.Context(), tx, tenantID)
	if err != nil {
		return response.InternalError(c, "Failed to list categories: "+err.Error())
	}

	return response.Success(c, categories)
}

// Search performs full-text search.
func (h *Handler) Search(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	query := c.Query("q")
	if query == "" {
		return response.BadRequest(c, "Search query 'q' parameter is required")
	}

	language := c.Query("language", "en")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	offset := (page - 1) * limit

	results, err := h.service.SearchArticles(c.Context(), tx, query, language, limit, offset)
	if err != nil {
		return response.InternalError(c, "Search failed: "+err.Error())
	}

	return response.Success(c, results)
}

// GetHomeFeed aggregates all essential home blocks in a single, fast response.
func (h *Handler) GetHomeFeed(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	tenantID := 1
	if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	} else if sess := middleware.SessionFromCtx(c); sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	}

	language := c.Query("language", "en")
	districtSlug := c.Query("district")

	homeData, err := h.service.GetHomeFeed(c.Context(), tx, tenantID, language, districtSlug)
	if err != nil {
		return response.InternalError(c, "Failed to load home feed: "+err.Error())
	}

	return response.Success(c, homeData)
}

// GetPersonalizedFeed returns user-personalized content.
func (h *Handler) GetPersonalizedFeed(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	sess := middleware.SessionFromCtx(c)

	filter := ListArticlesFilter{
		Status:   "published",
		Language: c.Query("language", "en"),
		SortBy:   "latest",
		Page:     c.QueryInt("page", 1),
		PerPage:  c.QueryInt("per_page", 20),
	}

	if sess != nil && sess.ActiveTenantID > 0 {
		// Scoped by user active tenant edition
	}

	articles, total, err := h.service.ListArticles(c.Context(), tx, filter)
	if err != nil {
		return response.InternalError(c, "Failed to fetch feed")
	}

	return response.Paginated(c, articles, filter.Page, filter.PerPage, total)
}

// ─── Studio Handlers ────────────────────────────

// StudioListArticles returns all articles across workflow statuses (for Kanban).
func (h *Handler) StudioListArticles(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)

	filter := ListArticlesFilter{
		Status:   c.Query("status"),
		Language: c.Query("language"),
		Category: c.Query("category"),
		Page:     c.QueryInt("page", 1),
		PerPage:  c.QueryInt("per_page", 50),
	}

	articles, total, err := h.service.ListArticles(c.Context(), tx, filter)
	if err != nil {
		return response.InternalError(c, "Failed to list articles: "+err.Error())
	}

	return response.Paginated(c, articles, filter.Page, filter.PerPage, total)
}

// CreateArticle creates a new draft article.
func (h *Handler) CreateArticle(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var input CreateArticleInput
	if err := c.BodyParser(&input); err != nil || input.Title == "" {
		return response.BadRequest(c, "Valid article title is required")
	}

	article, err := h.service.CreateArticle(c.Context(), tx, int(sess.ActiveTenantID), sess.UserID, input)
	if err != nil {
		return response.InternalError(c, "Failed to create article: "+err.Error())
	}

	return response.Created(c, article)
}

// GetArticle returns an article by ID.
func (h *Handler) GetArticle(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	article, err := h.service.GetArticle(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to get article: "+err.Error())
	}
	if article == nil {
		return response.NotFound(c, "Article not found")
	}

	return response.Success(c, article)
}

// TransitionStatus transitions article status in the editorial workflow.
func (h *Handler) TransitionStatus(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return response.BadRequest(c, "Target status is required")
	}

	if err := h.service.TransitionStatus(c.Context(), tx, id, body.Status, sess.UserID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Article status transitioned successfully",
		"status":  body.Status,
	})
}

// GetArticleVersions returns edit history.
func (h *Handler) GetArticleVersions(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	versions, err := h.service.GetArticleVersions(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to fetch versions: "+err.Error())
	}

	return response.Success(c, versions)
}

// ─── Stories & Live Blogs ───────────────────────

func (h *Handler) CreateStory(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	var body struct {
		Slug string `json:"slug"`
	}
	_ = c.BodyParser(&body)

	storyID, err := h.service.CreateStory(c.Context(), tx, int(sess.ActiveTenantID), body.Slug)
	if err != nil {
		return response.InternalError(c, "Failed to create story: "+err.Error())
	}

	return response.Created(c, fiber.Map{"story_id": storyID})
}

func (h *Handler) GetStoryVariants(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	storyID, err := uuid.Parse(c.Params("storyId"))
	if err != nil {
		return response.BadRequest(c, "Invalid story ID format")
	}

	variants, err := h.service.GetStoryVariants(c.Context(), tx, storyID)
	if err != nil {
		return response.InternalError(c, "Failed to get story variants: "+err.Error())
	}

	return response.Success(c, variants)
}

func (h *Handler) AddLiveBlogEntry(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	var body struct {
		Body     string `json:"body"`
		IsPinned bool   `json:"is_pinned"`
	}
	if err := c.BodyParser(&body); err != nil || body.Body == "" {
		return response.BadRequest(c, "Entry body is required")
	}

	entry, err := h.service.AddLiveBlogEntry(c.Context(), tx, id, sess.UserID, []byte(body.Body), body.IsPinned)
	if err != nil {
		return response.InternalError(c, "Failed to add live blog entry: "+err.Error())
	}

	return response.Created(c, entry)
}

func (h *Handler) ListLiveBlogEntries(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := uuid.Parse(c.Params("articleId"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	entries, err := h.service.ListLiveBlogEntries(c.Context(), tx, id)
	if err != nil {
		return response.InternalError(c, "Failed to list live blog entries: "+err.Error())
	}

	return response.Success(c, entries)
}
