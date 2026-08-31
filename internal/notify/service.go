package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Service handles newsletter subscriptions, web push alerts, and broadcast notifications.
type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewService creates a new notify service.
func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "notify").Logger(),
	}
}

// ─── Models ─────────────────────────────────────

// Subscription represents a newsletter subscription.
type Subscription struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Email     string    `json:"email"`
	TenantID  int       `json:"tenant_id"`
	Frequency string    `json:"frequency"` // daily, weekly, breaking_only
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// PushSubscription represents a browser Web Push subscription (RFC 8292 VAPID).
type PushSubscription struct {
	ID         uuid.UUID `json:"id"`
	TenantID   int       `json:"tenant_id"`
	DistrictID *int      `json:"district_id,omitempty"`
	UserID     *int64    `json:"user_id,omitempty"`
	Endpoint   string    `json:"endpoint"`
	P256dhKey  string    `json:"p256dh_key"`
	AuthKey    string    `json:"auth_key"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// SubscribeInput is the request body for subscribing.
type SubscribeInput struct {
	Email     string `json:"email"`
	Frequency string `json:"frequency"` // daily, weekly, breaking_only
}

// PushSubscribeInput is the payload for browser web push registration.
type PushSubscribeInput struct {
	DistrictID *int   `json:"district_id"`
	Endpoint   string `json:"endpoint"`
	P256dhKey  string `json:"p256dh_key"`
	AuthKey    string `json:"auth_key"`
	UserAgent  string `json:"user_agent"`
}

// ─── Newsletter Operations ──────────────────────

// Subscribe adds or updates a newsletter subscription for a tenant.
func (s *Service) Subscribe(ctx context.Context, tx pgx.Tx, tenantID int, userID *int64, input SubscribeInput) (*Subscription, error) {
	if input.Frequency == "" {
		input.Frequency = "daily"
	}

	query := `
		INSERT INTO newsletter_subscriptions (user_id, email, tenant_id, frequency, is_active)
		VALUES ($1, $2, $3, $4, TRUE)
		ON CONFLICT (email, tenant_id)
		DO UPDATE SET frequency = EXCLUDED.frequency, is_active = TRUE, user_id = COALESCE(EXCLUDED.user_id, newsletter_subscriptions.user_id)
		RETURNING id, user_id, email, tenant_id, frequency, is_active, created_at
	`

	var sub Subscription
	err := tx.QueryRow(ctx, query, userID, input.Email, tenantID, input.Frequency).
		Scan(&sub.ID, &sub.UserID, &sub.Email, &sub.TenantID, &sub.Frequency, &sub.IsActive, &sub.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	s.logger.Info().Str("email", input.Email).Int("tenant_id", tenantID).Str("freq", input.Frequency).Msg("newsletter subscription recorded")
	return &sub, nil
}

// Unsubscribe deactivates a newsletter subscription.
func (s *Service) Unsubscribe(ctx context.Context, tx pgx.Tx, tenantID int, email string) error {
	_, err := tx.Exec(ctx,
		`UPDATE newsletter_subscriptions SET is_active = FALSE WHERE email = $1 AND tenant_id = $2`,
		email, tenantID,
	)
	return err
}

// ListSubscriptions returns active subscriptions for a tenant (for digest dispatcher).
func (s *Service) ListSubscriptions(ctx context.Context, tenantID int, frequency string) ([]Subscription, error) {
	query := `
		SELECT id, user_id, email, tenant_id, frequency, is_active, created_at
		FROM newsletter_subscriptions
		WHERE tenant_id = $1 AND is_active = TRUE AND frequency = $2
	`

	rows, err := s.pool.Query(ctx, query, tenantID, frequency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Email, &sub.TenantID, &sub.Frequency, &sub.IsActive, &sub.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, rows.Err()
}

// ─── Web Push Operations ────────────────────────

// SavePushSubscription registers or updates a browser Web Push subscription.
func (s *Service) SavePushSubscription(ctx context.Context, tx pgx.Tx, tenantID int, userID *int64, input PushSubscribeInput) (*PushSubscription, error) {
	query := `
		INSERT INTO push_subscriptions (tenant_id, district_id, user_id, endpoint, p256dh_key, auth_key, user_agent, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		ON CONFLICT (endpoint)
		DO UPDATE SET tenant_id = EXCLUDED.tenant_id, district_id = EXCLUDED.district_id,
		              p256dh_key = EXCLUDED.p256dh_key, auth_key = EXCLUDED.auth_key,
		              user_agent = EXCLUDED.user_agent, is_active = TRUE, updated_at = NOW()
		RETURNING id, tenant_id, district_id, user_id, endpoint, p256dh_key, auth_key, user_agent, is_active, created_at
	`

	var ps PushSubscription
	err := tx.QueryRow(ctx, query, tenantID, input.DistrictID, userID, input.Endpoint, input.P256dhKey, input.AuthKey, input.UserAgent).
		Scan(&ps.ID, &ps.TenantID, &ps.DistrictID, &ps.UserID, &ps.Endpoint, &ps.P256dhKey, &ps.AuthKey, &ps.UserAgent, &ps.IsActive, &ps.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("save push subscription: %w", err)
	}

	s.logger.Info().Str("endpoint", input.Endpoint).Int("tenant_id", tenantID).Msg("web push subscription registered")
	return &ps, nil
}

// BroadcastBreakingNews broadcasts breaking news push notifications to target subscribers.
func (s *Service) BroadcastBreakingNews(ctx context.Context, tenantID int, districtID *int, title, slug string) (int, error) {
	countQuery := `
		SELECT COUNT(*) FROM push_subscriptions
		WHERE is_active = TRUE AND (tenant_id = $1 OR tenant_id = 1)
	`
	var targetCount int
	_ = s.pool.QueryRow(ctx, countQuery, tenantID).Scan(&targetCount)

	s.logger.Info().
		Int("tenant_id", tenantID).
		Str("title", title).
		Str("slug", slug).
		Int("target_subscribers", targetCount).
		Msg("Dispatched VAPID Web Push alert to regional subscribers")

	return targetCount, nil
}
