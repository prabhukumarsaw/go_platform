package content

import (
	"context"
	"strings"
	"time"
)

// TrendingSearchItem represents a trending search term with its search frequency.
type TrendingSearchItem struct {
	Term  string `json:"term"`
	Count int64  `json:"count"`
}

// CleanSearchTerm standardizes and validates a search query for analytics.
func CleanSearchTerm(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.Trim(cleaned, "\"'.,!?-:;()[]{}")
	if len([]rune(cleaned)) < 2 || len([]rune(cleaned)) > 60 {
		return ""
	}
	return cleaned
}

// RecordSearchQuery records a search query in Redis sorted sets.
func (s *Service) RecordSearchQuery(ctx context.Context, rawTerm string) {
	term := CleanSearchTerm(rawTerm)
	if term == "" || s.redis == nil {
		return
	}

	today := time.Now().Format("20060102")
	globalKey := "trending_searches:global"
	dailyKey := "trending_searches:daily:" + today

	pipe := s.redis.Pipeline()
	pipe.ZIncrBy(ctx, globalKey, 1, term)
	pipe.ZIncrBy(ctx, dailyKey, 1, term)
	pipe.Expire(ctx, dailyKey, 7*24*time.Hour)
	_, _ = pipe.Exec(ctx)
}

// GetTrendingSearches retrieves only genuine recorded trending search queries from Redis.
// Returns empty slice if no searches have been recorded yet.
func (s *Service) GetTrendingSearches(ctx context.Context, limit int) []TrendingSearchItem {
	if limit <= 0 || limit > 30 {
		limit = 14
	}

	results := make([]TrendingSearchItem, 0)
	if s.redis == nil {
		return results
	}

	vals, err := s.redis.ZRevRangeWithScores(ctx, "trending_searches:global", 0, int64(limit-1)).Result()
	if err != nil || len(vals) == 0 {
		return results
	}

	for _, z := range vals {
		termStr, ok := z.Member.(string)
		if !ok || termStr == "" {
			continue
		}
		results = append(results, TrendingSearchItem{
			Term:  termStr,
			Count: int64(z.Score),
		})
	}

	return results
}
