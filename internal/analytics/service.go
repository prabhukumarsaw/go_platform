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

func (s *Service) GetOverview(ctx context.Context, tx pgx.Tx, authorID *int64) (*AnalyticsOverview, error) {
	var overview AnalyticsOverview

	// Aggregate counts with optional author filtering
	countQuery := "SELECT COUNT(*), COUNT(*) FILTER (WHERE status='published'), COUNT(*) FILTER (WHERE status='draft'), COUNT(*) FILTER (WHERE status IN ('review', 'approved')), COALESCE(SUM(view_count), 0), COUNT(*) FILTER (WHERE is_breaking=TRUE) FROM articles"
	if authorID != nil {
		countQuery += " WHERE author_id = $1"
		_ = tx.QueryRow(ctx, countQuery, *authorID).
			Scan(&overview.TotalArticles, &overview.TotalPublished, &overview.TotalDrafts, &overview.TotalReview, &overview.TotalViews, &overview.TotalBreaking)
	} else {
		_ = tx.QueryRow(ctx, countQuery).
			Scan(&overview.TotalArticles, &overview.TotalPublished, &overview.TotalDrafts, &overview.TotalReview, &overview.TotalViews, &overview.TotalBreaking)
	}

	_ = tx.QueryRow(ctx, "SELECT COUNT(*) FROM newsletter_subscriptions WHERE is_active = TRUE").Scan(&overview.TotalSubscribers)

	// Regional / Category readership distribution
	stateQuery := `
		SELECT c.id, c.name, COALESCE(SUM(a.view_count), 0) as views, COUNT(a.id) as articles
		FROM categories c
		LEFT JOIN article_categories ac ON ac.category_id = c.id
		LEFT JOIN articles a ON a.id = ac.article_id AND a.status = 'published'
		WHERE c.level = 1 OR c.parent_id IS NULL
	`
	if authorID != nil {
		stateQuery += " AND (a.author_id = $1 OR a.author_id IS NULL)"
	}
	stateQuery += `
		GROUP BY c.id, c.name
		ORDER BY views DESC
		LIMIT 10
	`
	var rows pgx.Rows
	var err error
	if authorID != nil {
		rows, err = tx.Query(ctx, stateQuery, *authorID)
	} else {
		rows, err = tx.Query(ctx, stateQuery)
	}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tr RegionalReadership
			if err := rows.Scan(&tr.RegionID, &tr.RegionName, &tr.Views, &tr.Articles); err == nil {
				overview.StateDistribution = append(overview.StateDistribution, tr)
			}
		}
	}

	// Category breakdown
	catQuery := `
		SELECT c.name, COUNT(ac.article_id) as count
		FROM categories c
		LEFT JOIN article_categories ac ON ac.category_id = c.id
		LEFT JOIN articles a ON a.id = ac.article_id
		WHERE 1=1
	`
	if authorID != nil {
		catQuery += " AND (a.author_id = $1 OR a.author_id IS NULL)"
	}
	catQuery += `
		GROUP BY c.name
		ORDER BY count DESC
		LIMIT 8
	`
	var cRows pgx.Rows
	var cErr error
	if authorID != nil {
		cRows, cErr = tx.Query(ctx, catQuery, *authorID)
	} else {
		cRows, cErr = tx.Query(ctx, catQuery)
	}
	if cErr == nil {
		defer cRows.Close()
		for cRows.Next() {
			var cr CategoryReadership
			if cErr := cRows.Scan(&cr.CategoryName, &cr.Count); cErr == nil {
				overview.CategoryBreakdown = append(overview.CategoryBreakdown, cr)
			}
		}
	}

	// Top trending articles
	trendQuery := `
		SELECT id, title, slug, view_count, language
		FROM articles
		WHERE status = 'published'
	`
	if authorID != nil {
		trendQuery += " AND author_id = $1"
	}
	trendQuery += `
		ORDER BY view_count DESC, published_at DESC
		LIMIT 5
	`
	var tRows pgx.Rows
	var tErr error
	if authorID != nil {
		tRows, tErr = tx.Query(ctx, trendQuery, *authorID)
	} else {
		tRows, tErr = tx.Query(ctx, trendQuery)
	}
	if tErr == nil {
		defer tRows.Close()
		for tRows.Next() {
			var ts TrendingStat
			if tErr := tRows.Scan(&ts.ID, &ts.Title, &ts.Slug, &ts.ViewCount, &ts.Language); tErr == nil {
				overview.TopTrendingArticles = append(overview.TopTrendingArticles, ts)
			}
		}
	}

	return &overview, nil
}

func (s *Service) GetAuthorLeaderboard(ctx context.Context, tx pgx.Tx) ([]AuthorLeaderboard, error) {
	query := `
		SELECT u.id, u.display_name, 'National Newsroom Desk' as bureau_name,
		       COALESCE(SUM(a.view_count), 0) as total_views,
		       COUNT(a.id) as articles
		FROM users u
		JOIN articles a ON a.author_id = u.id AND a.status = 'published'
		WHERE u.is_staff = TRUE
		GROUP BY u.id, u.display_name
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
		if err := rows.Scan(&al.AuthorID, &al.DisplayName, &al.BureauName, &al.TotalViews, &al.Articles); err != nil {
			return nil, err
		}
		list = append(list, al)
	}
	return list, rows.Err()
}
