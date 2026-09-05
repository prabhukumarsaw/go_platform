package webstory

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

type WebStory struct {
	ID          uuid.UUID       `json:"id"`
	Title       string          `json:"title"`
	Slug        string          `json:"slug"`
	Language    string          `json:"language"`
	CoverImage  string          `json:"cover_image"`
	Slides      json.RawMessage `json:"slides"`
	AuthorID    int64           `json:"author_id"`
	Status      string          `json:"status"`
	ViewCount   int64           `json:"view_count"`
	PublishedAt time.Time       `json:"published_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "webstory").Logger(),
	}
}

func (s *Service) ListWebStories(ctx context.Context, tx pgx.Tx, language string, limit, offset int) ([]WebStory, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if language == "" {
		language = "hi"
	}

	query := `
		SELECT id, title, slug, language, cover_image, slides, author_id, status, view_count, published_at, created_at
		FROM web_stories
		WHERE status = 'published' AND language = $1
		ORDER BY published_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := tx.Query(ctx, query, language, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []WebStory
	for rows.Next() {
		var st WebStory
		if err := rows.Scan(
			&st.ID, &st.Title, &st.Slug, &st.Language,
			&st.CoverImage, &st.Slides, &st.AuthorID, &st.Status,
			&st.ViewCount, &st.PublishedAt, &st.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, st)
	}
	return list, rows.Err()
}

func (s *Service) GetWebStoryBySlug(ctx context.Context, tx pgx.Tx, slug, language string) (*WebStory, error) {
	query := `
		SELECT id, title, slug, language, cover_image, slides, author_id, status, view_count, published_at, created_at
		FROM web_stories
		WHERE slug = $1 AND language = $2 AND status = 'published'
	`

	var st WebStory
	err := tx.QueryRow(ctx, query, slug, language).Scan(
		&st.ID, &st.Title, &st.Slug, &st.Language,
		&st.CoverImage, &st.Slides, &st.AuthorID, &st.Status,
		&st.ViewCount, &st.PublishedAt, &st.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Increment view count
	go func() {
		_, _ = s.pool.Exec(context.Background(), "UPDATE web_stories SET view_count = view_count + 1 WHERE id = $1", st.ID)
	}()

	return &st, nil
}

func (s *Service) CreateWebStory(ctx context.Context, tx pgx.Tx, authorID int64, title, language, coverImage string, slides json.RawMessage) (*WebStory, error) {
	slug := generateSlug(title)
	if language == "" {
		language = "hi"
	}

	query := `
		INSERT INTO web_stories (title, slug, language, cover_image, slides, author_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, slug, language, cover_image, slides, author_id, status, view_count, published_at, created_at
	`

	var st WebStory
	err := tx.QueryRow(ctx, query, title, slug, language, coverImage, slides, authorID).Scan(
		&st.ID, &st.Title, &st.Slug, &st.Language,
		&st.CoverImage, &st.Slides, &st.AuthorID, &st.Status,
		&st.ViewCount, &st.PublishedAt, &st.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create web story: %w", err)
	}
	return &st, nil
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == ' ' {
			return r
		}
		return -1
	}, slug)
	slug = strings.Join(strings.Fields(slug), "-")
	if len(slug) > 180 {
		slug = slug[:180]
	}
	slug += "-" + uuid.New().String()[:6]
	return slug
}
