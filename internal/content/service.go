package content

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service handles article, story, and content operations.
type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewService creates a new content service.
func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "content").Logger(),
	}
}

// ─── Models ─────────────────────────────────────

// Article represents a full article with metadata.
type Article struct {
	ID              uuid.UUID        `json:"id"`
	StoryID         *uuid.UUID       `json:"story_id,omitempty"`
	TenantID        int              `json:"tenant_id"`
	DistrictID      *int             `json:"district_id,omitempty"`
	DistrictName    string           `json:"district_name,omitempty"`
	Language        string           `json:"language"`
	Title           string           `json:"title"`
	Slug            string           `json:"slug"`
	Body            json.RawMessage  `json:"body"`
	Excerpt         *string          `json:"excerpt,omitempty"`
	Status          string           `json:"status"`
	AuthorID        int64            `json:"author_id"`
	EditorID        *int64           `json:"editor_id,omitempty"`
	ReviewerID      *int64           `json:"reviewer_id,omitempty"`
	IsBreaking      bool             `json:"is_breaking"`
	IsFeatured      bool             `json:"is_featured"`
	IsNational      bool             `json:"is_national"`
	PublishedAt     *time.Time       `json:"published_at,omitempty"`
	ScheduledAt     *time.Time       `json:"scheduled_at,omitempty"`
	MetaTitle       string           `json:"meta_title"`
	MetaDescription string           `json:"meta_description"`
	OGImage         string           `json:"og_image"`
	FeaturedImage   string           `json:"featured_image"`
	ViewCount       int64            `json:"view_count"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	AuthorName      string           `json:"author_name,omitempty"`
	CategoryNames   []string         `json:"categories,omitempty"`
	TagNames        []string         `json:"tags,omitempty"`
}

// ArticleListItem is a lighter article representation for list views.
type ArticleListItem struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Excerpt       *string    `json:"excerpt,omitempty"`
	Status        string     `json:"status"`
	Language      string     `json:"language"`
	AuthorID      int64      `json:"author_id"`
	AuthorName    string     `json:"author_name"`
	IsBreaking    bool       `json:"is_breaking"`
	IsFeatured    bool       `json:"is_featured"`
	IsNational    bool       `json:"is_national"`
	FeaturedImage string     `json:"featured_image"`
	ViewCount     int64      `json:"view_count"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CategoryNames []string   `json:"categories,omitempty"`
	TagNames      []string   `json:"tags,omitempty"`
}

// Category represents a news category taxonomy.
type Category struct {
	ID        int    `json:"id"`
	TenantID  int    `json:"tenant_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SortOrder int    `json:"sort_order"`
}

// LiveBlogEntry represents an append-only live update.
type LiveBlogEntry struct {
	ID        int64           `json:"id"`
	ArticleID uuid.UUID       `json:"article_id"`
	Body      json.RawMessage `json:"body"`
	AuthorID  int64           `json:"author_id"`
	IsPinned  bool            `json:"is_pinned"`
	CreatedAt time.Time       `json:"created_at"`
}

// ArticleVersion represents edit history.
type ArticleVersion struct {
	ID         int64           `json:"id"`
	ArticleID  uuid.UUID       `json:"article_id"`
	EditedBy   int64           `json:"edited_by"`
	Diff       json.RawMessage `json:"diff"`
	VersionNum int             `json:"version_num"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CreateArticleInput is the input for creating a new article.
type CreateArticleInput struct {
	StoryID         *uuid.UUID      `json:"story_id"`
	Title           string          `json:"title"`
	Language        string          `json:"language"`
	Body            json.RawMessage `json:"body"`
	Excerpt         string          `json:"excerpt"`
	DistrictID      *int            `json:"district_id"`
	IsBreaking      bool            `json:"is_breaking"`
	IsFeatured      bool            `json:"is_featured"`
	IsNational      bool            `json:"is_national"`
	MetaTitle       string          `json:"meta_title"`
	MetaDescription string          `json:"meta_description"`
	FeaturedImage   string          `json:"featured_image"`
	CategoryIDs     []int           `json:"category_ids"`
	TagIDs          []int           `json:"tag_ids"`
}

// ─── Article CRUD ───────────────────────────────

// CreateArticle creates a new article draft.
func (s *Service) CreateArticle(ctx context.Context, tx pgx.Tx, tenantID int, authorID int64, input CreateArticleInput) (*Article, error) {
	slug := generateSlug(input.Title)

	if input.Language == "" {
		input.Language = "en"
	}

	query := `
		INSERT INTO articles
			(story_id, tenant_id, district_id, language, title, slug, body, excerpt,
			 status, author_id, is_breaking, is_featured, is_national,
			 meta_title, meta_description, featured_image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'draft', $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, story_id, tenant_id, district_id, language, title, slug, body, excerpt,
				  status, author_id, is_breaking, is_featured, is_national,
				  meta_title, meta_description, featured_image,
				  view_count, created_at, updated_at
	`

	var a Article
	err := tx.QueryRow(ctx, query,
		input.StoryID, tenantID, input.DistrictID, input.Language, input.Title, slug, input.Body, nilIfEmpty(input.Excerpt),
		authorID, input.IsBreaking, input.IsFeatured, input.IsNational,
		input.MetaTitle, input.MetaDescription, input.FeaturedImage,
	).Scan(
		&a.ID, &a.StoryID, &a.TenantID, &a.DistrictID, &a.Language, &a.Title, &a.Slug, &a.Body, &a.Excerpt,
		&a.Status, &a.AuthorID, &a.IsBreaking, &a.IsFeatured, &a.IsNational,
		&a.MetaTitle, &a.MetaDescription, &a.FeaturedImage,
		&a.ViewCount, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert article: %w", err)
	}

	// Link categories
	for _, catID := range input.CategoryIDs {
		_, _ = tx.Exec(ctx, "INSERT INTO article_categories (article_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", a.ID, catID)
	}

	// Link tags
	for _, tagID := range input.TagIDs {
		_, _ = tx.Exec(ctx, "INSERT INTO article_tags (article_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", a.ID, tagID)
	}

	// Record initial version
	_, _ = tx.Exec(ctx, `INSERT INTO article_versions (article_id, edited_by, diff, snapshot, version_num) VALUES ($1, $2, '{}', $3, 1)`, a.ID, authorID, input.Body)

	return &a, nil
}

// GetArticle retrieves a single article by ID.
func (s *Service) GetArticle(ctx context.Context, tx pgx.Tx, articleID uuid.UUID) (*Article, error) {
	query := `
		SELECT a.id, a.story_id, a.tenant_id, a.district_id, a.language,
			   a.title, a.slug, a.body, a.excerpt, a.status,
			   a.author_id, a.editor_id, a.reviewer_id,
			   a.is_breaking, a.is_featured, a.is_national,
			   a.published_at, a.scheduled_at,
			   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), COALESCE(a.og_image, ''), COALESCE(a.featured_image, ''),
			   a.view_count, a.created_at, a.updated_at,
			   COALESCE(u.display_name, '') as author_name
		FROM articles a
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.id = $1
	`

	var a Article
	err := tx.QueryRow(ctx, query, articleID).Scan(
		&a.ID, &a.StoryID, &a.TenantID, &a.DistrictID, &a.Language,
		&a.Title, &a.Slug, &a.Body, &a.Excerpt, &a.Status,
		&a.AuthorID, &a.EditorID, &a.ReviewerID,
		&a.IsBreaking, &a.IsFeatured, &a.IsNational,
		&a.PublishedAt, &a.ScheduledAt,
		&a.MetaTitle, &a.MetaDescription, &a.OGImage, &a.FeaturedImage,
		&a.ViewCount, &a.CreatedAt, &a.UpdatedAt,
		&a.AuthorName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}

	return &a, nil
}

// GetArticleBySlug retrieves a published article by slug (for reader).
func (s *Service) GetArticleBySlug(ctx context.Context, tx pgx.Tx, slug, language string) (*Article, error) {
	query := `
		SELECT a.id, a.story_id, a.tenant_id, a.district_id, a.language,
			   a.title, a.slug, a.body, a.excerpt, a.status,
			   a.author_id, a.editor_id, a.reviewer_id,
			   a.is_breaking, a.is_featured, a.is_national,
			   a.published_at, a.scheduled_at,
			   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), COALESCE(a.og_image, ''), COALESCE(a.featured_image, ''),
			   a.view_count, a.created_at, a.updated_at,
			   COALESCE(u.display_name, '') as author_name
		FROM articles a
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.slug = $1 AND a.language = $2 AND a.status = 'published'
	`

	var a Article
	err := tx.QueryRow(ctx, query, slug, language).Scan(
		&a.ID, &a.StoryID, &a.TenantID, &a.DistrictID, &a.Language,
		&a.Title, &a.Slug, &a.Body, &a.Excerpt, &a.Status,
		&a.AuthorID, &a.EditorID, &a.ReviewerID,
		&a.IsBreaking, &a.IsFeatured, &a.IsNational,
		&a.PublishedAt, &a.ScheduledAt,
		&a.MetaTitle, &a.MetaDescription, &a.OGImage, &a.FeaturedImage,
		&a.ViewCount, &a.CreatedAt, &a.UpdatedAt,
		&a.AuthorName,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get article by slug: %w", err)
	}

	// Increment view count
	go func() {
		_, _ = s.pool.Exec(context.Background(), "UPDATE articles SET view_count = view_count + 1 WHERE id = $1", a.ID)
	}()

	return &a, nil
}

// ListArticlesFilter options with full filter matrix.
type ListArticlesFilter struct {
	TenantID     int         `query:"tenant_id"`
	Status       string      `query:"status"`
	Language     string      `query:"language"`
	Category     string      `query:"category"`
	Categories   []string    `query:"categories"` // Support multi-category matching (e.g. ['jharkhand', 'health'])
	Tags         []string    `query:"tags"`       // Support multi-tag matching
	ExcludeIDs   []uuid.UUID `query:"exclude_ids"`
	DistrictSlug string      `query:"district"`
	AuthorID     *int64      `query:"author_id"`
	IsBreaking   *bool       `query:"is_breaking"`
	IsFeatured   *bool       `query:"is_featured"`
	Period       string      `query:"period"` // today, yesterday, week, month
	DateFrom     *time.Time  `query:"from"`
	DateTo       *time.Time  `query:"to"`
	SortBy       string      `query:"sort"` // latest (default), trending, views, oldest
	Page         int         `query:"page"`
	PerPage      int         `query:"per_page"`
}

// ListArticles queries articles with filtering and pagination.
func (s *Service) ListArticles(ctx context.Context, tx pgx.Tx, filter ListArticlesFilter) ([]ArticleListItem, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}

	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Language != "" {
		conditions = append(conditions, fmt.Sprintf("a.language = $%d", argIdx))
		args = append(args, filter.Language)
		argIdx++
	}
	if filter.AuthorID != nil {
		conditions = append(conditions, fmt.Sprintf("a.author_id = $%d", argIdx))
		args = append(args, *filter.AuthorID)
		argIdx++
	}
	if filter.IsBreaking != nil {
		conditions = append(conditions, fmt.Sprintf("a.is_breaking = $%d", argIdx))
		args = append(args, *filter.IsBreaking)
		argIdx++
	}
	if filter.IsFeatured != nil {
		conditions = append(conditions, fmt.Sprintf("a.is_featured = $%d", argIdx))
		args = append(args, *filter.IsFeatured)
		argIdx++
	}

	// Single or multi-category filtering
	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM article_categories ac
				JOIN categories c ON c.id = ac.category_id
				WHERE ac.article_id = a.id AND (c.slug = $%d OR c.name ILIKE $%d)
			)
		`, argIdx, argIdx))
		args = append(args, filter.Category)
		argIdx++
	} else if len(filter.Categories) > 0 {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM article_categories ac
				JOIN categories c ON c.id = ac.category_id
				WHERE ac.article_id = a.id AND (c.slug = ANY($%d) OR c.name = ANY($%d))
			)
		`, argIdx, argIdx))
		args = append(args, filter.Categories)
		argIdx++
	}

	// Tags filter
	if len(filter.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM article_tags at
				JOIN tags t ON t.id = at.tag_id
				WHERE at.article_id = a.id AND (t.slug = ANY($%d) OR t.name = ANY($%d))
			)
		`, argIdx, argIdx))
		args = append(args, filter.Tags)
		argIdx++
	}

	// Exclude already seen IDs (Strict visual deduplication)
	if len(filter.ExcludeIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("a.id != ALL($%d)", argIdx))
		args = append(args, filter.ExcludeIDs)
		argIdx++
	}

	// District filter via slug join
	if filter.DistrictSlug != "" {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM districts d
				WHERE d.id = a.district_id AND d.slug = $%d
			)
		`, argIdx))
		args = append(args, filter.DistrictSlug)
		argIdx++
	}

	// Date / Period filters
	now := time.Now()
	if filter.Period == "today" {
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		conditions = append(conditions, fmt.Sprintf("a.published_at >= $%d", argIdx))
		args = append(args, startOfDay)
		argIdx++
	} else if filter.Period == "yesterday" {
		yesterday := now.AddDate(0, 0, -1)
		startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		endOfYesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		conditions = append(conditions, fmt.Sprintf("a.published_at >= $%d AND a.published_at < $%d", argIdx, argIdx+1))
		args = append(args, startOfYesterday, endOfYesterday)
		argIdx += 2
	} else if filter.Period == "week" {
		conditions = append(conditions, fmt.Sprintf("a.published_at >= $%d", argIdx))
		args = append(args, now.AddDate(0, 0, -7))
		argIdx++
	}

	// Custom date range
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("a.published_at >= $%d", argIdx))
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("a.published_at <= $%d", argIdx))
		args = append(args, *filter.DateTo)
		argIdx++
	}

	if filter.TenantID > 1 {
		conditions = append(conditions, fmt.Sprintf("(a.tenant_id = $%d OR a.is_national = TRUE)", argIdx))
		args = append(args, filter.TenantID)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a WHERE %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}

	// Ordering logic (prioritizes local state dispatches first, then national news)
	orderBy := "a.published_at DESC NULLS LAST"
	if filter.TenantID > 1 {
		orderBy = fmt.Sprintf("CASE WHEN a.tenant_id = %d THEN 0 ELSE 1 END, a.published_at DESC NULLS LAST", filter.TenantID)
	}
	if filter.SortBy == "trending" || filter.SortBy == "views" {
		orderBy = "a.view_count DESC, a.published_at DESC"
	} else if filter.SortBy == "oldest" {
		orderBy = "a.published_at ASC"
	} else if filter.Status != "published" {
		orderBy = "a.updated_at DESC"
	}

	offset := (filter.Page - 1) * filter.PerPage
	listQuery := fmt.Sprintf(`
		SELECT a.id, a.title, a.slug, a.excerpt, a.status, a.language,
			   a.author_id, COALESCE(u.display_name, '') as author_name,
			   a.is_breaking, a.is_featured, a.is_national, COALESCE(a.featured_image, ''),
			   a.view_count, a.published_at, a.created_at, a.updated_at,
			   COALESCE(ARRAY_AGG(DISTINCT c.name) FILTER (WHERE c.name IS NOT NULL), '{}') as category_names,
			   COALESCE(ARRAY_AGG(DISTINCT t.name) FILTER (WHERE t.name IS NOT NULL), '{}') as tag_names
		FROM articles a
		LEFT JOIN users u ON u.id = a.author_id
		LEFT JOIN article_categories ac ON ac.article_id = a.id
		LEFT JOIN categories c ON c.id = ac.category_id
		LEFT JOIN article_tags at ON at.article_id = a.id
		LEFT JOIN tags t ON t.id = at.tag_id
		WHERE %s
		GROUP BY a.id, u.display_name
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderBy, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := tx.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list articles: %w", err)
	}
	defer rows.Close()

	var articles []ArticleListItem
	for rows.Next() {
		var a ArticleListItem
		if err := rows.Scan(
			&a.ID, &a.Title, &a.Slug, &a.Excerpt, &a.Status, &a.Language,
			&a.AuthorID, &a.AuthorName,
			&a.IsBreaking, &a.IsFeatured, &a.IsNational, &a.FeaturedImage,
			&a.ViewCount, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt,
			&a.CategoryNames, &a.TagNames,
		); err != nil {
			return nil, 0, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}

	return articles, total, rows.Err()
}

// ListCategories returns all categories available for a tenant.
func (s *Service) ListCategories(ctx context.Context, tx pgx.Tx, tenantID int) ([]Category, error) {
	query := `SELECT id, tenant_id, name, slug, sort_order FROM categories WHERE tenant_id = $1 ORDER BY sort_order, name`
	rows, err := tx.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Slug, &c.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// ─── Status Transitions ─────────────────────────

var validTransitions = map[string][]string{
	"draft":     {"review"},
	"review":    {"approved", "draft"},
	"approved":  {"published", "draft"},
	"published": {"archived", "draft"},
	"archived":  {"draft"},
}

// TransitionStatus changes an article's editorial status.
func (s *Service) TransitionStatus(ctx context.Context, tx pgx.Tx, articleID uuid.UUID, newStatus string, userID int64) error {
	var currentStatus string
	err := tx.QueryRow(ctx, "SELECT status FROM articles WHERE id = $1", articleID).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("fetch article status: %w", err)
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return fmt.Errorf("no transitions available from status '%s'", currentStatus)
	}

	valid := false
	for _, st := range allowed {
		if st == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid transition: %s → %s", currentStatus, newStatus)
	}

	updateQuery := "UPDATE articles SET status = $1, updated_at = NOW()"
	args := []interface{}{newStatus}
	argIdx := 2

	if newStatus == "published" {
		updateQuery += fmt.Sprintf(", published_at = NOW(), editor_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if newStatus == "approved" || newStatus == "review" {
		updateQuery += fmt.Sprintf(", reviewer_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}

	updateQuery += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, articleID)

	_, err = tx.Exec(ctx, updateQuery, args...)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	_, _ = tx.Exec(ctx,
		`INSERT INTO article_versions (article_id, edited_by, diff) VALUES ($1, $2, $3)`,
		articleID, userID, fmt.Sprintf(`{"transition":"%s->%s"}`, currentStatus, newStatus),
	)

	return nil
}

// ─── Stories (Multi-Language Variant Linking) ───

func (s *Service) CreateStory(ctx context.Context, tx pgx.Tx, tenantID int, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, "INSERT INTO stories (tenant_id) VALUES ($1) RETURNING id", tenantID).Scan(&id)
	return id, err
}

func (s *Service) GetStoryVariants(ctx context.Context, tx pgx.Tx, storyID uuid.UUID) ([]ArticleListItem, error) {
	query := `
		SELECT a.id, a.title, a.slug, a.excerpt, a.status, a.language,
			   a.author_id, COALESCE(u.display_name, '') as author_name,
			   a.is_breaking, a.is_featured, a.is_national, COALESCE(a.featured_image, ''),
			   a.view_count, a.published_at, a.created_at, a.updated_at
		FROM articles a
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.story_id = $1
	`
	rows, err := tx.Query(ctx, query, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ArticleListItem
	for rows.Next() {
		var a ArticleListItem
		if err := rows.Scan(
			&a.ID, &a.Title, &a.Slug, &a.Excerpt, &a.Status, &a.Language,
			&a.AuthorID, &a.AuthorName,
			&a.IsBreaking, &a.IsFeatured, &a.IsNational, &a.FeaturedImage,
			&a.ViewCount, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// ─── Live Blog Entries ──────────────────────────

func (s *Service) AddLiveBlogEntry(ctx context.Context, tx pgx.Tx, articleID uuid.UUID, authorID int64, body json.RawMessage, isPinned bool) (*LiveBlogEntry, error) {
	query := `
		INSERT INTO live_blog_entries (article_id, body, author_id, is_pinned)
		VALUES ($1, $2, $3, $4)
		RETURNING id, article_id, body, author_id, is_pinned, created_at
	`
	var e LiveBlogEntry
	err := tx.QueryRow(ctx, query, articleID, body, authorID, isPinned).
		Scan(&e.ID, &e.ArticleID, &e.Body, &e.AuthorID, &e.IsPinned, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("add live blog entry: %w", err)
	}
	return &e, nil
}

func (s *Service) ListLiveBlogEntries(ctx context.Context, tx pgx.Tx, articleID uuid.UUID) ([]LiveBlogEntry, error) {
	query := `
		SELECT id, article_id, body, author_id, is_pinned, created_at
		FROM live_blog_entries
		WHERE article_id = $1
		ORDER BY is_pinned DESC, created_at DESC
	`
	rows, err := tx.Query(ctx, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []LiveBlogEntry
	for rows.Next() {
		var e LiveBlogEntry
		if err := rows.Scan(&e.ID, &e.ArticleID, &e.Body, &e.AuthorID, &e.IsPinned, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// ─── Version History ────────────────────────────

func (s *Service) GetArticleVersions(ctx context.Context, tx pgx.Tx, articleID uuid.UUID) ([]ArticleVersion, error) {
	query := `
		SELECT id, article_id, edited_by, diff, version_num, created_at
		FROM article_versions
		WHERE article_id = $1
		ORDER BY created_at DESC
	`
	rows, err := tx.Query(ctx, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ArticleVersion
	for rows.Next() {
		var v ArticleVersion
		if err := rows.Scan(&v.ID, &v.ArticleID, &v.EditedBy, &v.Diff, &v.VersionNum, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ─── Search with Multilingual Transliteration ──

var phoneticTransliterationMap = map[string][]string{
	"chandrayaan": {"चंद्रयान", "chandrayaan", "chandrayan", "lunar"},
	"chandrayan":  {"चंद्रयान", "chandrayaan", "lunar"},
	"चंद्रयान":     {"chandrayaan", "chandrayan", "चंद्रयान", "lunar"},
	"modi":        {"मोदी", "modi"},
	"मोदी":        {"modi", "मोदी"},
	"cricket":     {"क्रिकेट", "cricket"},
	"क्रिकेट":     {"cricket", "क्रिकेट"},
	"election":    {"चुनाव", "election", "chunav"},
	"chunav":      {"चुनाव", "election"},
	"चुनाव":        {"election", "chunav", "चुनाव"},
	"mumbai":      {"मुंबई", "mumbai"},
	"मुंबई":        {"mumbai", "मुंबई"},
	"isro":        {"इसरो", "isro"},
	"इसरो":        {"isro", "इसरो"},
	"budget":      {"बजट", "budget"},
	"बजट":         {"budget", "बजट"},
	"farmer":      {"किसान", "farmer", "kisan"},
	"kisan":       {"किसान", "farmer"},
	"किसान":        {"farmer", "kisan", "किसान"},
	"delhi":       {"दिल्ली", "delhi"},
	"दिल्ली":       {"delhi", "दिल्ली"},
	"jharkhand":   {"झारखंड", "jharkhand"},
	"झारखंड":       {"jharkhand", "झारखंड"},
	"maharashtra": {"महाराष्ट्र", "maharashtra"},
	"महाराष्ट्र":   {"maharashtra", "महाराष्ट्र"},
	"india":       {"भारत", "india", "bharat"},
	"bharat":      {"भारत", "india"},
	"भारत":        {"india", "bharat", "भारत"},
}

func expandSearchTerms(term string) []string {
	normalized := strings.ToLower(strings.TrimSpace(term))
	if terms, found := phoneticTransliterationMap[normalized]; found {
		return terms
	}
	return []string{normalized}
}

func (s *Service) SearchArticles(ctx context.Context, tx pgx.Tx, term, language string, limit, offset int) ([]ArticleListItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	searchVariants := expandSearchTerms(term)

	// Build dynamic ILIKE conditions for all phonetic transliterations
	var conditions []string
	var args []interface{}
	argIdx := 1

	for _, v := range searchVariants {
		conditions = append(conditions, fmt.Sprintf("(a.title ILIKE $%d OR a.excerpt ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+v+"%")
		argIdx++
	}

	searchClause := strings.Join(conditions, " OR ")

	query := fmt.Sprintf(`
		SELECT a.id, a.title, a.slug, a.excerpt, a.status, a.language,
			   a.author_id, COALESCE(u.display_name, '') as author_name,
			   a.is_breaking, a.is_featured, a.is_national, COALESCE(a.featured_image, ''),
			   a.view_count, a.published_at, a.created_at, a.updated_at
		FROM articles a
		LEFT JOIN users u ON u.id = a.author_id
		WHERE a.status = 'published'
		  AND (%s)
		ORDER BY a.published_at DESC
		LIMIT $%d OFFSET $%d
	`, searchClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []ArticleListItem{}
	for rows.Next() {
		var a ArticleListItem
		if err := rows.Scan(
			&a.ID, &a.Title, &a.Slug, &a.Excerpt, &a.Status, &a.Language,
			&a.AuthorID, &a.AuthorName,
			&a.IsBreaking, &a.IsFeatured, &a.IsNational, &a.FeaturedImage,
			&a.ViewCount, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// ─── Helpers ────────────────────────────────────

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == ' ' {
			return r
		}
		return -1
	}, slug)
	slug = strings.Join(strings.Fields(slug), "-")
	if len(slug) > 200 {
		slug = slug[:200]
	}
	slug += "-" + uuid.New().String()[:8]
	return slug
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
