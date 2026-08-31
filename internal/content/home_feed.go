package content

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// HomeFeedResponse is the consolidated, unified home feed payload.
type HomeFeedResponse struct {
	StateEdition         StateEditionInfo             `json:"state_edition"`
	Breaking             []ArticleListItem            `json:"breaking"`
	Featured             []ArticleListItem            `json:"featured"`
	Latest               []ArticleListItem            `json:"latest"`
	StateNews            []ArticleListItem            `json:"state_news"`
	Trending             []ArticleListItem            `json:"trending"`
	Recommendations      []ArticleListItem            `json:"recommendations"`
	CategorySections     map[string][]ArticleListItem `json:"category_sections"`
	WebStories           []interface{}                `json:"web_stories"`
	ActivePoll           interface{}                  `json:"active_poll,omitempty"`
	EPapers              []interface{}                `json:"epapers"`
	TopAuthors           []AuthorSpotlight            `json:"top_authors"`
	NavigationCategories []Category                   `json:"navigation_categories"`
}

// StateEditionInfo contains active state metadata.
type StateEditionInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// AuthorSpotlight contains author byline summary.
type AuthorSpotlight struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Role        string `json:"role"`
	StoryCount  int    `json:"story_count"`
}

// GetHomeFeed aggregates all essential homepage news blocks in a single, lightning-fast call.
func (s *Service) GetHomeFeed(ctx context.Context, tx pgx.Tx, tenantID int, language, districtSlug string) (*HomeFeedResponse, error) {
	if language == "" {
		language = "en"
	}

	resp := &HomeFeedResponse{
		CategorySections: make(map[string][]ArticleListItem),
		Breaking:         []ArticleListItem{},
		Featured:         []ArticleListItem{},
		Latest:           []ArticleListItem{},
		StateNews:        []ArticleListItem{},
		Trending:         []ArticleListItem{},
		Recommendations:  []ArticleListItem{},
		WebStories:       []interface{}{},
		EPapers:          []interface{}{},
		TopAuthors:       []AuthorSpotlight{},
	}

	// 1. State info
	var stateName, stateSlug string
	_ = tx.QueryRow(ctx, "SELECT name, slug FROM tenants WHERE id = $1", tenantID).Scan(&stateName, &stateSlug)
	resp.StateEdition = StateEditionInfo{
		ID:   tenantID,
		Name: stateName,
		Slug: stateSlug,
	}

	// 2. Navigation Categories
	categories, err := s.ListCategories(ctx, tx, tenantID)
	if err == nil {
		resp.NavigationCategories = categories
	} else {
		resp.NavigationCategories = []Category{}
	}

	// Dynamic deduplication map
	seen := make(map[string]bool)
	getExcluded := func() []string {
		var list []string
		for id := range seen {
			list = append(list, id)
		}
		return list
	}
	_ = getExcluded

	// 3. Breaking News (limit 5)
	breakingFilter := ListArticlesFilter{
		TenantID: tenantID,
		Language: language,
		Status:   "published",
		PerPage:  5,
	}
	isBreaking := true
	breakingFilter.IsBreaking = &isBreaking
	if items, _, err := s.ListArticles(ctx, tx, breakingFilter); err == nil && len(items) > 0 {
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				resp.Breaking = append(resp.Breaking, item)
			}
		}
	}

	// 4. Featured Spotlight (limit 4) - Deduplicated
	featuredFilter := ListArticlesFilter{
		TenantID: tenantID,
		Language: language,
		Status:   "published",
		PerPage:  8,
	}
	isFeatured := true
	featuredFilter.IsFeatured = &isFeatured
	if items, _, err := s.ListArticles(ctx, tx, featuredFilter); err == nil && len(items) > 0 {
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				resp.Featured = append(resp.Featured, item)
				if len(resp.Featured) >= 4 {
					break
				}
			}
		}
	}

	// 5. State / Regional News (prioritizes local tenant articles) - Deduplicated
	stateFilter := ListArticlesFilter{
		TenantID:     tenantID,
		Language:     language,
		DistrictSlug: districtSlug,
		Status:       "published",
		PerPage:      12,
	}
	if items, _, err := s.ListArticles(ctx, tx, stateFilter); err == nil {
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				resp.StateNews = append(resp.StateNews, item)
				if len(resp.StateNews) >= 6 {
					break
				}
			}
		}
	}

	// 6. Latest Stream (limit 12) - Deduplicated
	latestFilter := ListArticlesFilter{
		TenantID: 1, // National/Global feed
		Language: language,
		Status:   "published",
		SortBy:   "latest",
		PerPage:  20,
	}
	if items, _, err := s.ListArticles(ctx, tx, latestFilter); err == nil {
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				resp.Latest = append(resp.Latest, item)
				if len(resp.Latest) >= 12 {
					break
				}
			}
		}
	}

	// 7. Trending News (limit 6) - Deduplicated (Global / National trending rank)
	trendingFilter := ListArticlesFilter{
		TenantID: 1,
		Language: language,
		Status:   "published",
		SortBy:   "trending",
		PerPage:  12,
	}
	if items, _, err := s.ListArticles(ctx, tx, trendingFilter); err == nil {
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				resp.Trending = append(resp.Trending, item)
				if len(resp.Trending) >= 5 {
					break
				}
			}
		}
		// If all were seen in earlier blocks, allow top trending stories
		if len(resp.Trending) == 0 && len(items) > 0 {
			resp.Trending = items[:min(len(items), 5)]
		}
	}

	// 8. Recommendations / Similar news pool - Deduplicated
	recFilter := ListArticlesFilter{
		TenantID: 1,
		Language: language,
		Status:   "published",
		PerPage:  12,
	}
	if items, _, err := s.ListArticles(ctx, tx, recFilter); err == nil {
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				resp.Recommendations = append(resp.Recommendations, item)
				if len(resp.Recommendations) >= 6 {
					break
				}
			}
		}
		// If all were seen in earlier blocks, allow top recommendation stories
		if len(resp.Recommendations) == 0 && len(items) > 0 {
			resp.Recommendations = items[:min(len(items), 4)]
		}
	}

	// 9. Key Dynamic Category Sections (politics, business, technology, sports, entertainment, health, crime)
	targetCats := []string{"politics", "business", "technology", "sports", "entertainment", "health", "crime"}
	for _, cat := range targetCats {
		catFilter := ListArticlesFilter{
			TenantID: tenantID,
			Language: language,
			Category: cat,
			Status:   "published",
			PerPage:  6,
		}
		items, _, _ := s.ListArticles(ctx, tx, catFilter)
		var catArticles []ArticleListItem
		for _, item := range items {
			if !seen[item.ID.String()] {
				seen[item.ID.String()] = true
				catArticles = append(catArticles, item)
				if len(catArticles) >= 4 {
					break
				}
			}
		}
		// If specific category has no distinct tags left, allow top items
		if len(catArticles) == 0 && len(items) > 0 {
			catArticles = items[:min(len(items), 3)]
		}
		resp.CategorySections[cat] = catArticles
	}

	// 10. Top Authors / Columnists
	authorQuery := `
		SELECT u.id, u.display_name, COALESCE(u.avatar_url, ''), COUNT(a.id) as story_count
		FROM users u
		JOIN articles a ON a.author_id = u.id
		WHERE a.status = 'published'
		GROUP BY u.id, u.display_name, u.avatar_url
		ORDER BY story_count DESC
		LIMIT 5
	`
	if aRows, err := tx.Query(ctx, authorQuery); err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var auth AuthorSpotlight
			if scanErr := aRows.Scan(&auth.ID, &auth.DisplayName, &auth.AvatarURL, &auth.StoryCount); scanErr == nil {
				auth.Role = "Senior Bureau Chief"
				resp.TopAuthors = append(resp.TopAuthors, auth)
			}
		}
	}

	// 11. Active Poll & E-Papers & Web Stories
	pollQuery := `
		SELECT id, question, total_votes
		FROM polls
		WHERE is_active = TRUE AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`
	var pollID, question string
	var totalVotes int64
	if err := tx.QueryRow(ctx, pollQuery).Scan(&pollID, &question, &totalVotes); err == nil {
		resp.ActivePoll = map[string]interface{}{
			"id":          pollID,
			"question":    question,
			"total_votes": totalVotes,
		}
	}

	// Web stories query
	wsQuery := `
		SELECT id, title, slug, cover_image, jsonb_array_length(slides) as slide_count
		FROM web_stories
		WHERE status = 'published'
		ORDER BY published_at DESC NULLS LAST
		LIMIT 8
	`
	if wsRows, err := tx.Query(ctx, wsQuery); err == nil {
		defer wsRows.Close()
		for wsRows.Next() {
			var wsID, title, slug, coverImage string
			var slideCount int
			if scanErr := wsRows.Scan(&wsID, &title, &slug, &coverImage, &slideCount); scanErr == nil {
				resp.WebStories = append(resp.WebStories, map[string]interface{}{
					"id":          wsID,
					"title":       title,
					"slug":        slug,
					"cover_image": coverImage,
					"slide_count": slideCount,
				})
			}
		}
	}

	// E-Papers query
	epQuery := `
		SELECT id, title, edition_date, COALESCE(thumbnail_url, ''), page_count
		FROM epapers
		WHERE is_active = TRUE
		ORDER BY edition_date DESC
		LIMIT 4
	`
	if epRows, err := tx.Query(ctx, epQuery); err == nil {
		defer epRows.Close()
		for epRows.Next() {
			var epID, epTitle, thumbURL string
			var epDate interface{}
			var pageCount int
			if scanErr := epRows.Scan(&epID, &epTitle, &epDate, &thumbURL, &pageCount); scanErr == nil {
				resp.EPapers = append(resp.EPapers, map[string]interface{}{
					"id":            epID,
					"title":         epTitle,
					"thumbnail_url": thumbURL,
					"page_count":    pageCount,
				})
			}
		}
	}

	return resp, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
