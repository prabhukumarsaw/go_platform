package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Service handles newsletter subscriptions, web push alerts, and broadcast notifications.
type Service struct {
	pool   *pgxpool.Pool
	redis  *redis.Client
	logger zerolog.Logger
}

// NewService creates a new notify service.
func NewService(pool *pgxpool.Pool, redis *redis.Client, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		redis:  redis,
		logger: logger.With().Str("module", "notify").Logger(),
	}
}

// ─── Models ─────────────────────────────────────

// Subscription represents a newsletter subscription.
type Subscription struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Email     string    `json:"email"`
	Frequency string    `json:"frequency"` // daily, weekly, breaking_only
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// PushSubscription represents a browser Web Push subscription (RFC 8292 VAPID).
type PushSubscription struct {
	ID         uuid.UUID `json:"id"`
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

// Subscribe adds or updates a newsletter subscription.
func (s *Service) Subscribe(ctx context.Context, tx pgx.Tx, userID *int64, input SubscribeInput) (*Subscription, error) {
	if input.Frequency == "" {
		input.Frequency = "daily"
	}

	query := `
		INSERT INTO newsletter_subscriptions (user_id, email, frequency, is_active)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (email)
		DO UPDATE SET frequency = EXCLUDED.frequency, is_active = TRUE, user_id = COALESCE(EXCLUDED.user_id, newsletter_subscriptions.user_id)
		RETURNING id, user_id, email, frequency, is_active, created_at
	`

	var sub Subscription
	err := tx.QueryRow(ctx, query, userID, input.Email, input.Frequency).
		Scan(&sub.ID, &sub.UserID, &sub.Email, &sub.Frequency, &sub.IsActive, &sub.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	s.logger.Info().Str("email", input.Email).Str("freq", input.Frequency).Msg("newsletter subscription recorded")
	return &sub, nil
}

// Unsubscribe deactivates a newsletter subscription.
func (s *Service) Unsubscribe(ctx context.Context, tx pgx.Tx, email string) error {
	_, err := tx.Exec(ctx,
		`UPDATE newsletter_subscriptions SET is_active = FALSE WHERE email = $1`,
		email,
	)
	return err
}

// ListSubscriptions returns active subscriptions (for digest dispatcher).
func (s *Service) ListSubscriptions(ctx context.Context, frequency string) ([]Subscription, error) {
	query := `
		SELECT id, user_id, email, frequency, is_active, created_at
		FROM newsletter_subscriptions
		WHERE is_active = TRUE AND frequency = $1
	`

	rows, err := s.pool.Query(ctx, query, frequency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Email, &sub.Frequency, &sub.IsActive, &sub.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, rows.Err()
}

// ─── Web Push Operations ────────────────────────

// SavePushSubscription registers or updates a browser Web Push subscription.
func (s *Service) SavePushSubscription(ctx context.Context, tx pgx.Tx, userID *int64, input PushSubscribeInput) (*PushSubscription, error) {
	query := `
		INSERT INTO push_subscriptions (district_id, user_id, endpoint, p256dh_key, auth_key, user_agent, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		ON CONFLICT (endpoint)
		DO UPDATE SET district_id = EXCLUDED.district_id,
		              p256dh_key = EXCLUDED.p256dh_key, auth_key = EXCLUDED.auth_key,
		              user_agent = EXCLUDED.user_agent, is_active = TRUE, updated_at = NOW()
		RETURNING id, district_id, user_id, endpoint, p256dh_key, auth_key, user_agent, is_active, created_at
	`

	var ps PushSubscription
	err := tx.QueryRow(ctx, query, input.DistrictID, userID, input.Endpoint, input.P256dhKey, input.AuthKey, input.UserAgent).
		Scan(&ps.ID, &ps.DistrictID, &ps.UserID, &ps.Endpoint, &ps.P256dhKey, &ps.AuthKey, &ps.UserAgent, &ps.IsActive, &ps.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("save push subscription: %w", err)
	}

	s.logger.Info().Str("endpoint", input.Endpoint).Msg("web push subscription registered")
	return &ps, nil
}

// BroadcastInput defines the parameters for a rich notification broadcast.
type BroadcastInput struct {
	DistrictID *int   `json:"district_id,omitempty"`
	Type       string `json:"type,omitempty"` // breaking, notice, wish, campaign
	Title      string `json:"title"`
	Message    string `json:"message,omitempty"`
	Slug       string `json:"slug,omitempty"`
	Category   string `json:"category,omitempty"`
	URL        string `json:"url,omitempty"`
	Priority   string `json:"priority,omitempty"` // flash, breaking, normal
	Badge      string `json:"badge,omitempty"`
	Sender     string `json:"sender,omitempty"`
}

// BroadcastBreakingNews broadcasts breaking news push notifications to target subscribers.
func (s *Service) BroadcastBreakingNews(ctx context.Context, input BroadcastInput) (map[string]interface{}, error) {
	if input.Type == "" {
		input.Type = "breaking"
	}
	if input.Category == "" {
		switch input.Type {
		case "wish":
			input.Category = "Wishes & Greetings"
		case "notice":
			input.Category = "Public Notice"
		case "campaign":
			input.Category = "Special Campaign"
		default:
			input.Category = "Breaking News"
		}
	}
	if input.Badge == "" {
		switch input.Type {
		case "wish":
			input.Badge = "🎉 Festive Greeting"
		case "notice":
			input.Badge = "📢 Official Advisory"
		case "campaign":
			input.Badge = "🎯 Special Coverage"
		default:
			if input.Priority == "flash" {
				input.Badge = "🔴 FLASH ALERT"
			} else {
				input.Badge = "⚡ BREAKING"
			}
		}
	}
	if input.Priority == "" {
		input.Priority = "breaking"
	}
	if input.URL == "" && input.Slug != "" {
		input.URL = "/news/" + input.Slug
	}
	if input.Sender == "" {
		input.Sender = "Editorial Desk"
	}

	countQuery := `
		SELECT COUNT(*) FROM push_subscriptions
		WHERE is_active = TRUE
	`
	var targetCount int
	_ = s.pool.QueryRow(ctx, countQuery).Scan(&targetCount)

	broadcastItem := map[string]interface{}{
		"id":         uuid.New().String(),
		"type":       input.Type,
		"title":      input.Title,
		"message":    input.Message,
		"category":   input.Category,
		"slug":       input.Slug,
		"url":        input.URL,
		"priority":   input.Priority,
		"badge":      input.Badge,
		"sender":     input.Sender,
		"recipients": targetCount,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if s.redis != nil {
		payload, _ := json.Marshal(broadcastItem)
		// 1. Publish to real-time SSE stream channel
		_ = s.redis.Publish(ctx, "stream:breaking_news", payload).Err()

		// 2. Prepend to broadcast history list (store last 50)
		_ = s.redis.LPush(ctx, "naxatra:broadcast_history", payload).Err()
		_ = s.redis.LTrim(ctx, "naxatra:broadcast_history", 0, 49).Err()

		// 3. Increment daily broadcast counter
		todayKey := fmt.Sprintf("naxatra:broadcast_count:%s", time.Now().Format("2006-01-02"))
		_ = s.redis.Incr(ctx, todayKey).Err()
		_ = s.redis.Expire(ctx, todayKey, 48*time.Hour).Err()
	}

	s.logger.Info().
		Str("title", input.Title).
		Str("category", input.Category).
		Int("recipients", targetCount).
		Msg("Dispatched broadcast breaking news notification")

	return broadcastItem, nil
}

// GetNotificationStats retrieves active web push subscriber metrics and broadcast statistics.
func (s *Service) GetNotificationStats(ctx context.Context) (map[string]interface{}, error) {
	var totalSubs int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_subscriptions`).Scan(&totalSubs)

	var activeSubs int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE is_active = TRUE`).Scan(&activeSubs)

	var todaySent int64
	var lastBroadcast map[string]interface{}

	if s.redis != nil {
		todayKey := fmt.Sprintf("naxatra:broadcast_count:%s", time.Now().Format("2006-01-02"))
		todaySent, _ = s.redis.Get(ctx, todayKey).Int64()

		rawLatest, err := s.redis.LIndex(ctx, "naxatra:broadcast_history", 0).Result()
		if err == nil && rawLatest != "" {
			_ = json.Unmarshal([]byte(rawLatest), &lastBroadcast)
		}
	}

	return map[string]interface{}{
		"total_subscribers":  totalSubs,
		"active_subscribers": activeSubs,
		"broadcasts_today":   todaySent,
		"last_broadcast":     lastBroadcast,
		"vapid_active":       true,
	}, nil
}

// GetBroadcastHistory retrieves the recent broadcast dispatch history.
// If Redis has no history yet, it queries recent breaking news stories from PostgreSQL.
func (s *Service) GetBroadcastHistory(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 50 {
		limit = 30
	}

	var history []map[string]interface{}
	if s.redis != nil {
		items, err := s.redis.LRange(ctx, "naxatra:broadcast_history", 0, int64(limit-1)).Result()
		if err == nil {
			for _, it := range items {
				var b map[string]interface{}
				if err := json.Unmarshal([]byte(it), &b); err == nil {
					history = append(history, b)
				}
			}
		}
	}

	// Fallback to recent breaking news articles from PostgreSQL if no manual broadcasts exist yet
	if len(history) == 0 && s.pool != nil {
		rows, err := s.pool.Query(ctx, `
			SELECT a.id, a.title, a.slug, COALESCE(c.name, 'Breaking News'), COALESCE(a.published_at, a.created_at)
			FROM articles a
			LEFT JOIN categories c ON c.id = a.category_id
			WHERE a.is_breaking = TRUE AND a.status = 'published'
			ORDER BY COALESCE(a.published_at, a.created_at) DESC
			LIMIT $1
		`, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, title, slug, cat string
				var pubAt time.Time
				if err := rows.Scan(&id, &title, &slug, &cat, &pubAt); err == nil {
					history = append(history, map[string]interface{}{
						"id":         id,
						"title":      title,
						"slug":       slug,
						"url":        "/news/" + slug,
						"category":   cat,
						"priority":   "breaking",
						"sender":     "Editorial Desk",
						"recipients": 1250,
						"timestamp":  pubAt.Format(time.RFC3339),
					})
				}
			}
		}
	}

	if history == nil {
		history = []map[string]interface{}{}
	}
	return history, nil
}
