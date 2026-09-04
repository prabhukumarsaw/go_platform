package content

import (
	"encoding/json"
	"strconv"
	"strings"
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
	router.Get("/categories/tree", h.ListCategoriesTree)
	router.Get("/tags", h.ListTags)
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
	studio.Put("/:id", h.UpdateArticle)
	studio.Patch("/:id", h.UpdateArticle)
	studio.Post("/:id/transition", h.TransitionStatus)
	studio.Post("/:id/schedule", h.ScheduleArticle)
	studio.Get("/:id/versions", h.GetArticleVersions)

	// Studio Taxonomy CRUD
	router.Post("/studio/categories", h.CreateCategory)
	router.Delete("/studio/categories/:id", h.DeleteCategory)
	router.Post("/studio/tags", h.CreateTag)
	router.Delete("/studio/tags/:id", h.DeleteTag)

	// Studio Stories & Live blogs
	router.Post("/studio/stories", h.CreateStory)
	router.Post("/studio/live-blogs/:id/entries", h.AddLiveBlogEntry)
	router.Delete("/studio/live-blogs/entries/:id", h.DeleteLiveBlogEntry)
	router.Patch("/studio/live-blogs/entries/:id/pin", h.TogglePinLiveBlogEntry)
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
	sess := middleware.SessionFromCtx(c)
	tenantID := 1
	if sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	} else if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	}

	var categories []Category
	var err error
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		categories, err = h.service.ListCategories(c.Context(), tx, tenantID)
	} else {
		categories, err = h.service.ListCategoriesDirect(c.Context(), tenantID)
	}

	if err != nil {
		return response.InternalError(c, "Failed to list categories: "+err.Error())
	}

	return response.Success(c, categories)
}

// ListCategoriesTree returns categories formatted as a hierarchical parent-child tree.
func (h *Handler) ListCategoriesTree(c *fiber.Ctx) error {
	var roots []Category
	var err error

	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		roots, err = h.service.ListCategoriesTree(c.Context(), tx)
	} else {
		flat, dErr := h.service.ListCategoriesDirect(c.Context(), 1)
		if dErr != nil {
			return response.InternalError(c, "Failed to list categories: "+dErr.Error())
		}
		lookup := make(map[int]*Category)
		for i := range flat {
			flat[i].Children = []Category{}
			lookup[flat[i].ID] = &flat[i]
		}
		for i := range flat {
			node := lookup[flat[i].ID]
			if node.ParentID == nil || *node.ParentID == 0 {
				roots = append(roots, *node)
			} else if p, exists := lookup[*node.ParentID]; exists {
				p.Children = append(p.Children, *node)
			}
		}
		for i := range roots {
			if n, exists := lookup[roots[i].ID]; exists {
				roots[i] = *n
			}
		}
	}

	if err != nil {
		return response.InternalError(c, "Failed to list category tree: "+err.Error())
	}

	return response.Success(c, roots)
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
	tenantID := 1
	if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	} else if sess := middleware.SessionFromCtx(c); sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	}

	language := c.Query("language", "en")
	districtSlug := c.Query("district")

	var homeData *HomeFeedResponse
	var err error
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		homeData, err = h.service.GetHomeFeed(c.Context(), tx, tenantID, language, districtSlug)
	} else {
		homeData, err = h.service.GetHomeFeedDirect(c.Context(), tenantID, language, districtSlug)
	}
	if err != nil {
		return response.InternalError(c, "Failed to load home feed: "+err.Error())
	}

	// Cache hint for CDN / reverse proxies
	c.Set("Cache-Control", "public, max-age=30, stale-while-revalidate=60")
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

	status := c.Query("status")
	if status == "all" {
		status = ""
	}

	filter := ListArticlesFilter{
		Status:   status,
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

// UpdateArticle updates an existing article.
func (h *Handler) UpdateArticle(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	var input CreateArticleInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid article payload")
	}

	editorID := int64(1)
	if sess != nil && sess.UserID > 0 {
		editorID = sess.UserID
	}

	article, err := h.service.UpdateArticle(c.Context(), tx, id, editorID, input)
	if err != nil {
		return response.InternalError(c, "Failed to update article: "+err.Error())
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

// ScheduleArticle schedules an article to be published at a specified timestamp.
func (h *Handler) ScheduleArticle(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tx := c.Locals("tx").(pgx.Tx)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	var body struct {
		ScheduledAt string `json:"scheduled_at"`
	}
	if err := c.BodyParser(&body); err != nil || body.ScheduledAt == "" {
		return response.BadRequest(c, "scheduled_at timestamp is required")
	}

	scheduledTime, err := time.Parse(time.RFC3339, body.ScheduledAt)
	if err != nil {
		return response.BadRequest(c, "Invalid scheduled_at format (expected RFC3339, e.g. 2026-09-03T18:00:00Z)")
	}

	if err := h.service.ScheduleArticle(c.Context(), tx, id, scheduledTime, sess.UserID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message":      "Article scheduled successfully",
		"status":       "scheduled",
		"scheduled_at": scheduledTime,
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
	tx, _ := c.Locals("tx").(pgx.Tx)

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	var body struct {
		Headline   string `json:"headline"`
		Body       string `json:"body"`
		IsPinned   bool   `json:"is_pinned"`
		IsBreaking bool   `json:"is_breaking"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Body) == "" {
		return response.BadRequest(c, "Entry body is required")
	}

	// Correctly encode string as JSON bytes so Postgres JSONB does not fail with syntax error
	jsonBody, err := json.Marshal(body.Body)
	if err != nil {
		jsonBody = []byte(`""`)
	}

	authorID := int64(1)
	if sess != nil && sess.UserID > 0 {
		authorID = sess.UserID
	}

	var entry *LiveBlogEntry
	if tx != nil {
		entry, err = h.service.AddLiveBlogEntry(c.Context(), tx, id, authorID, body.Headline, jsonBody, body.IsPinned, body.IsBreaking)
	} else {
		entry, err = h.service.AddLiveBlogEntryDirect(c.Context(), id, authorID, body.Headline, jsonBody, body.IsPinned, body.IsBreaking)
	}
	if err != nil {
		return response.InternalError(c, "Failed to add live blog entry: "+err.Error())
	}

	return response.Created(c, entry)
}

func (h *Handler) ListLiveBlogEntries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("articleId"))
	if err != nil {
		return response.BadRequest(c, "Invalid article ID format")
	}

	var entries []LiveBlogEntry
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		entries, err = h.service.ListLiveBlogEntries(c.Context(), tx, id)
	} else {
		entries, err = h.service.ListLiveBlogEntriesDirect(c.Context(), id)
	}
	if err != nil {
		return response.InternalError(c, "Failed to list live blog entries: "+err.Error())
	}

	return response.Success(c, entries)
}

func (h *Handler) DeleteLiveBlogEntry(c *fiber.Ctx) error {
	tx, _ := c.Locals("tx").(pgx.Tx)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid live blog entry ID")
	}

	if err := h.service.DeleteLiveBlogEntry(c.Context(), tx, id); err != nil {
		return response.InternalError(c, "Failed to delete live blog entry: "+err.Error())
	}

	return response.Success(c, fiber.Map{"deleted": true})
}

func (h *Handler) TogglePinLiveBlogEntry(c *fiber.Ctx) error {
	tx, _ := c.Locals("tx").(pgx.Tx)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "Invalid live blog entry ID")
	}

	var body struct {
		IsPinned bool `json:"is_pinned"`
	}
	_ = c.BodyParser(&body)

	if err := h.service.TogglePinLiveBlogEntry(c.Context(), tx, id, body.IsPinned); err != nil {
		return response.InternalError(c, "Failed to update pin status: "+err.Error())
	}

	return response.Success(c, fiber.Map{"updated": true})
}

// ─── Taxonomy Handlers ──────────────────────────

func (h *Handler) CreateCategory(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	tenantID := 1
	if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	}

	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		return response.BadRequest(c, "Category name is required")
	}

	cat, err := h.service.CreateCategory(c.Context(), tx, tenantID, strings.TrimSpace(body.Name), strings.TrimSpace(body.Slug))
	if err != nil {
		return response.InternalError(c, "Failed to create category: "+err.Error())
	}

	return response.Created(c, cat)
}

func (h *Handler) DeleteCategory(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid category ID")
	}

	if err := h.service.DeleteCategory(c.Context(), tx, id); err != nil {
		return response.InternalError(c, "Failed to delete category: "+err.Error())
	}

	return response.Success(c, fiber.Map{"deleted": true})
}

func (h *Handler) ListTags(c *fiber.Ctx) error {
	sess := middleware.SessionFromCtx(c)
	tenantID := 1
	if sess != nil && sess.ActiveTenantID > 0 {
		tenantID = int(sess.ActiveTenantID)
	} else if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	}

	var tags []Tag
	var err error
	if tx, ok := c.Locals("tx").(pgx.Tx); ok && tx != nil {
		tags, err = h.service.ListTags(c.Context(), tx, tenantID)
	} else {
		tags, err = h.service.ListTagsDirect(c.Context(), tenantID)
	}

	if err != nil {
		return response.InternalError(c, "Failed to list tags: "+err.Error())
	}

	return response.Success(c, tags)
}

func (h *Handler) CreateTag(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	tenantID := 1
	if tid, ok := c.Locals("tenant_id").(int64); ok && tid > 0 {
		tenantID = int(tid)
	}

	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		return response.BadRequest(c, "Tag name is required")
	}

	t, err := h.service.CreateTag(c.Context(), tx, tenantID, strings.TrimSpace(body.Name), strings.TrimSpace(body.Slug))
	if err != nil {
		return response.InternalError(c, "Failed to create tag: "+err.Error())
	}

	return response.Created(c, t)
}

func (h *Handler) DeleteTag(c *fiber.Ctx) error {
	tx := c.Locals("tx").(pgx.Tx)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid tag ID")
	}

	if err := h.service.DeleteTag(c.Context(), tx, id); err != nil {
		return response.InternalError(c, "Failed to delete tag: "+err.Error())
	}

	return response.Success(c, fiber.Map{"deleted": true})
}
