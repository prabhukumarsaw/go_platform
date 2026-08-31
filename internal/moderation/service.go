package moderation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service handles UGC and comment moderation.
type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewService creates a new moderation service.
func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "moderation").Logger(),
	}
}

// ─── Models ─────────────────────────────────────

// Comment represents a user comment on an article.
type Comment struct {
	ID          int64      `json:"id"`
	ArticleID   uuid.UUID  `json:"article_id"`
	UserID      int64      `json:"user_id"`
	UserName    string     `json:"user_name,omitempty"`
	UserAvatar  string     `json:"user_avatar,omitempty"`
	ParentID    *int64     `json:"parent_id,omitempty"`
	Body        string     `json:"body"`
	Status      string     `json:"status"` // pending, approved, rejected, spam
	ModeratedBy *int64     `json:"moderated_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// PostCommentInput is the payload to submit a comment.
type PostCommentInput struct {
	ArticleID uuid.UUID `json:"article_id"`
	ParentID  *int64    `json:"parent_id,omitempty"`
	Body      string    `json:"body"`
}

// ─── Operations ─────────────────────────────────

// AddComment submits a new comment (defaults to pending moderation).
func (s *Service) AddComment(ctx context.Context, tx pgx.Tx, userID int64, input PostCommentInput) (*Comment, error) {
	query := `
		INSERT INTO comments (article_id, user_id, parent_id, body, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id, article_id, user_id, parent_id, body, status, created_at
	`

	var c Comment
	err := tx.QueryRow(ctx, query, input.ArticleID, userID, input.ParentID, input.Body).
		Scan(&c.ID, &c.ArticleID, &c.UserID, &c.ParentID, &c.Body, &c.Status, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}

	return &c, nil
}

// ListApprovedComments returns approved comments for an article (public reader).
func (s *Service) ListApprovedComments(ctx context.Context, tx pgx.Tx, articleID uuid.UUID) ([]Comment, error) {
	query := `
		SELECT c.id, c.article_id, c.user_id, COALESCE(u.display_name, 'Reader'), COALESCE(u.avatar_url, ''),
		       c.parent_id, c.body, c.status, c.created_at
		FROM comments c
		LEFT JOIN users u ON u.id = c.user_id
		WHERE c.article_id = $1 AND c.status = 'approved'
		ORDER BY c.created_at ASC
	`

	rows, err := tx.Query(ctx, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ArticleID, &c.UserID, &c.UserName, &c.UserAvatar,
			&c.ParentID, &c.Body, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListPendingQueue returns unmoderated comments for moderation dashboard.
func (s *Service) ListPendingQueue(ctx context.Context, tx pgx.Tx, page, perPage int) ([]Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var total int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM comments WHERE status = 'pending'").Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := `
		SELECT c.id, c.article_id, c.user_id, COALESCE(u.display_name, 'Reader'), COALESCE(u.avatar_url, ''),
		       c.parent_id, c.body, c.status, c.created_at
		FROM comments c
		LEFT JOIN users u ON u.id = c.user_id
		WHERE c.status = 'pending'
		ORDER BY c.created_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := tx.Query(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ArticleID, &c.UserID, &c.UserName, &c.UserAvatar,
			&c.ParentID, &c.Body, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}

	return list, total, rows.Err()
}

// ModerateComment sets the status of a comment (approved, rejected, spam).
func (s *Service) ModerateComment(ctx context.Context, tx pgx.Tx, commentID int64, status string, moderatorID int64) error {
	if status != "approved" && status != "rejected" && status != "spam" {
		return fmt.Errorf("invalid moderation status: %s", status)
	}

	query := `
		UPDATE comments
		SET status = $1, moderated_by = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err := tx.Exec(ctx, query, status, moderatorID, commentID)
	return err
}

// ─── Reader Feedback & Correction Queue ─────────

type Feedback struct {
	ID        int64      `json:"id"`
	UserID    *int64     `json:"user_id,omitempty"`
	TenantID  int        `json:"tenant_id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Category  string     `json:"category"`
	ArticleID *uuid.UUID `json:"article_id,omitempty"`
	Message   string     `json:"message"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

type SubmitFeedbackInput struct {
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Category  string     `json:"category"`
	ArticleID *uuid.UUID `json:"article_id,omitempty"`
	Message   string     `json:"message"`
}

func (s *Service) SubmitFeedback(ctx context.Context, tx pgx.Tx, tenantID int, userID *int64, input SubmitFeedbackInput) (*Feedback, error) {
	if input.Category == "" {
		input.Category = "general"
	}
	query := `
		INSERT INTO feedbacks (tenant_id, user_id, name, email, category, article_id, message, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'open')
		RETURNING id, tenant_id, user_id, name, email, category, article_id, message, status, created_at
	`

	var fb Feedback
	err := tx.QueryRow(ctx, query, tenantID, userID, input.Name, input.Email, input.Category, input.ArticleID, input.Message).
		Scan(&fb.ID, &fb.TenantID, &fb.UserID, &fb.Name, &fb.Email, &fb.Category, &fb.ArticleID, &fb.Message, &fb.Status, &fb.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("submit feedback: %w", err)
	}

	return &fb, nil
}

func (s *Service) ListFeedbacks(ctx context.Context, tx pgx.Tx, tenantID int, status string) ([]Feedback, error) {
	query := `
		SELECT id, tenant_id, user_id, name, email, category, article_id, message, status, created_at
		FROM feedbacks
		WHERE (tenant_id = $1 OR $1 = 1)
	`
	args := []interface{}{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Feedback
	for rows.Next() {
		var fb Feedback
		if err := rows.Scan(&fb.ID, &fb.TenantID, &fb.UserID, &fb.Name, &fb.Email, &fb.Category, &fb.ArticleID, &fb.Message, &fb.Status, &fb.CreatedAt); err == nil {
			list = append(list, fb)
		}
	}
	return list, nil
}

func (s *Service) UpdateFeedbackStatus(ctx context.Context, tx pgx.Tx, feedbackID int64, status string) error {
	_, err := tx.Exec(ctx, "UPDATE feedbacks SET status = $1 WHERE id = $2", status, feedbackID)
	return err
}
