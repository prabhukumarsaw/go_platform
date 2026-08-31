package analytics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Service struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(pool *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		pool:   pool,
		logger: logger.With().Str("module", "analytics").Logger(),
	}
}

func (s *Service) GetOverview(ctx context.Context, tx pgx.Tx, tenantID int) (*AnalyticsOverview, error) {
	var overview AnalyticsOverview

	// Aggregate counts
	_ = tx.QueryRow(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE status='published'), COALESCE(SUM(view_count), 0), COUNT(*) FILTER (WHERE is_breaking=TRUE) FROM articles WHERE (tenant_id = $1 OR $1 = 1)", tenantID).
		Scan(&overview.TotalArticles, &overview.TotalPublished, &overview.TotalViews, &overview.TotalBreaking)

	_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM newsletter_subscriptions WHERE is_active = TRUE").Scan(&overview.TotalSubscribers)

	// State readership distribution
	stateQuery := `
		SELECT t.id, t.name, COALESCE(SUM(a.view_count), 0) as views, COUNT(a.id) as articles
		FROM tenants t
		LEFT JOIN articles a ON a.tenant_id = t.id AND a.status = 'published'
		GROUP BY t.id, t.name
		ORDER BY views DESC
		LIMIT 10
	`
	rows, err := tx.Query(ctx, stateQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tr TenantReadership
			if err := rows.Scan(&tr.TenantID, &tr.TenantName, &tr.Views, &tr.Articles); err == nil {
				overview.StateDistribution = append(overview.StateDistribution, tr)
			}
		}
	}

	// Category breakdown
	catQuery := `
		SELECT c.name, COUNT(ac.article_id) as count
		FROM categories c
		LEFT JOIN article_categories ac ON ac.category_id = c.id
		WHERE (c.tenant_id = $1 OR $1 = 1)
		GROUP BY c.name
		ORDER BY count DESC
		LIMIT 8
	`
	cRows, err := tx.Query(ctx, catQuery, tenantID)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var cr CategoryReadership
			if err := cRows.Scan(&cr.CategoryName, &cr.Count); err == nil {
				overview.CategoryBreakdown = append(overview.CategoryBreakdown, cr)
			}
		}
	}

	// Top trending articles
	trendQuery := `
		SELECT id::text, title, slug, view_count, language
		FROM articles
		WHERE status = 'published' AND (tenant_id = $1 OR $1 = 1)
		ORDER BY view_count DESC, published_at DESC
		LIMIT 5
	`
	tRows, err := tx.Query(ctx, trendQuery, tenantID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var ts TrendingStat
			if err := tRows.Scan(&ts.ID, &ts.Title, &ts.Slug, &ts.ViewCount, &ts.Language); err == nil {
				overview.TopTrendingArticles = append(overview.TopTrendingArticles, ts)
			}
		}
	}

	return &overview, nil
}

func (s *Service) GetAuthorLeaderboard(ctx context.Context, tx pgx.Tx) ([]AuthorLeaderboard, error) {
	query := `
		SELECT u.id, u.display_name, COALESCE(t.name, 'National'),
		       COALESCE(SUM(a.view_count), 0) as total_views,
		       COUNT(a.id) as articles
		FROM users u
		JOIN articles a ON a.author_id = u.id AND a.status = 'published'
		LEFT JOIN tenants t ON t.id = a.tenant_id
		WHERE u.is_staff = TRUE
		GROUP BY u.id, u.display_name, t.name
		ORDER BY total_views DESC
		LIMIT 15
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("author leaderboard: %w", err)
	}
	defer rows.Close()

	var list []AuthorLeaderboard
	for rows.Next() {
		var al AuthorLeaderboard
		if err := rows.Scan(&al.AuthorID, &al.DisplayName, &al.TenantName, &al.TotalViews, &al.Articles); err != nil {
			return nil, err
		}
		list = append(list, al)
	}
	return list, rows.Err()
}
