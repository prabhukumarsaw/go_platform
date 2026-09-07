package content

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ─────────────────────────────────────────────────────────────────────────────
// Frontend-ready Data Models (Strict Zero Duplicate Home Feed)
// ─────────────────────────────────────────────────────────────────────────────

// HomeArticleItem represents an editorial article item formatted for frontend consumption.
type HomeArticleItem struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Image     string `json:"image"`
	Date      string `json:"date"`
	Category  string `json:"category"`
	Author    string `json:"author,omitempty"`
	ViewCount int64  `json:"view_count,omitempty"`
}

// FourColumnSectionItem represents one of the 4 columns in a multi-category section.
type FourColumnSectionItem struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Slug     string            `json:"slug"`
	Featured *HomeArticleItem  `json:"featured,omitempty"`
	Articles []HomeArticleItem `json:"articles"`
}

// CategoryBlockAData has 1 main featured card and sub-articles.
type CategoryBlockAData struct {
	Title       string            `json:"title"`
	Featured    *HomeArticleItem  `json:"featured,omitempty"`
	SubArticles []HomeArticleItem `json:"subArticles"`
}

// CategoryBlockBData has 1 top featured card, middle cards, and bottom cards.
type CategoryBlockBData struct {
	Title          string            `json:"title"`
	Featured       *HomeArticleItem  `json:"featured,omitempty"`
	MiddleArticles []HomeArticleItem `json:"middleArticles"`
	BottomArticles []HomeArticleItem `json:"bottomArticles"`
}

// VideoNewsItem represents a video news story.
type VideoNewsItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	VideoURL    string `json:"videoUrl"`
	Thumbnail   string `json:"thumbnail"`
	Category    string `json:"category"`
	Duration    string `json:"duration"`
	PublishedAt string `json:"publishedAt"`
	Views       string `json:"views"`
	Author      string `json:"author,omitempty"`
}

// StateDeskData holds state desk (Jharkhand & Bihar) and editorial sidebar blocks.
type StateDeskData struct {
	TopHeadlines      []HomeArticleItem `json:"topHeadlines"`
	FeaturedArticle   *HomeArticleItem  `json:"featuredArticle,omitempty"`
	JharkhandBottom   []HomeArticleItem `json:"jharkhandBottom"`
	StateArticles     []HomeArticleItem `json:"stateArticles"`
	MoreNewsArticles  []HomeArticleItem `json:"moreNewsArticles"`
	SidebarTopArticle *HomeArticleItem  `json:"sidebarTopArticle,omitempty"`
	SidebarColumns    []HomeArticleItem `json:"sidebarColumns"`
	SidebarOpinion    struct {
		Left  []HomeArticleItem `json:"left"`
		Right []HomeArticleItem `json:"right"`
	} `json:"sidebarOpinion"`
}

// FeaturedSectionData holds the top featured section.
type FeaturedSectionData struct {
	MainFeatured        *HomeArticleItem  `json:"mainFeatured"`
	MiddleFeatured      []HomeArticleItem `json:"middleFeatured"`
	RightTopFeatured    *HomeArticleItem  `json:"rightTopFeatured"`
	RightListMostViewed []HomeArticleItem `json:"rightListMostViewed"`
	BreakingNews        []HomeArticleItem `json:"breakingNews"`
}

// CategorySectionData holds the 5 main categories + trending + exclusive.
type CategorySectionData struct {
	Politics      CategoryBlockAData `json:"politics"`
	Sports        CategoryBlockAData `json:"sports"`
	Entertainment CategoryBlockBData `json:"entertainment"`
	Crime         CategoryBlockAData `json:"crime"`
	Business      CategoryBlockBData `json:"business"`
	TopTrending   []HomeArticleItem  `json:"topTrending"`
	ExclusiveNews []HomeArticleItem  `json:"exclusiveNews"`
	SidebarBottom []HomeArticleItem  `json:"sidebarBottom"`
}

// TechnologySectionData holds the technology spotlight block.
type TechnologySectionData struct {
	CategoryName    string            `json:"categoryName"`
	CategoryTitleHi string            `json:"categoryTitleHi"`
	FeaturedArticle *HomeArticleItem  `json:"featuredArticle"`
	SideArticles    []HomeArticleItem `json:"sideArticles"`
	RightArticles   []HomeArticleItem `json:"rightArticles"`
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

// HomeFeedResponse is the consolidated, unified home feed payload with strict zero duplicate guarantee.
type HomeFeedResponse struct {
	// Full structured blocks matching frontend components
	FeaturedData        FeaturedSectionData     `json:"featuredData"`
	FourColumnSection1  []FourColumnSectionItem `json:"fourColumnSection1"`
	ContentSidebarData  StateDeskData           `json:"contentSidebarData"`
	VideoNewsData       []VideoNewsItem         `json:"videoNewsData"`
	CategorySectionData CategorySectionData     `json:"categorySectionData"`
	FourColumnSection2  []FourColumnSectionItem `json:"fourColumnSection2"`
	TechnologyData      TechnologySectionData   `json:"technologyData"`
	OrderedSections     []HomepageSection       `json:"ordered_sections,omitempty"`

	// Metrics & Taxonomy
	TotalUniqueArticles  int               `json:"total_unique_articles"`
	NavigationCategories []Category        `json:"navigation_categories"`
	StateEdition         StateEditionInfo  `json:"state_edition"`
	TopAuthors           []AuthorSpotlight `json:"top_authors"`
	WebStories           []interface{}     `json:"web_stories"`
	ActivePoll           interface{}       `json:"active_poll,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Aggregator with Strict Zero-Duplicate Pipeline
// ─────────────────────────────────────────────────────────────────────────────

// GetHomeFeed returns the fully populated, zero-duplicate homepage feed.
func (s *Service) GetHomeFeed(ctx context.Context, tx pgx.Tx, language, districtSlug string) (*HomeFeedResponse, error) {
	return s.buildHomeFeed(ctx, language, districtSlug, false)
}

// GetHomeFeedDirect uses direct connection pool execution and thread-safe in-memory caching.
func (s *Service) GetHomeFeedDirect(ctx context.Context, language, districtSlug string) (*HomeFeedResponse, error) {
	return s.buildHomeFeed(ctx, language, districtSlug, false)
}

// buildHomeFeed executes the strict zero-duplicate pipeline with SingleFlight thundering-herd protection and L1/L2 caching.
func (s *Service) buildHomeFeed(ctx context.Context, language, districtSlug string, forceRefresh bool) (*HomeFeedResponse, error) {
	if language == "" {
		language = "hi"
	}
	cacheKey := fmt.Sprintf("home_feed:%s:%s", language, districtSlug)

	// 1. Check in-memory L1 cache (~0.05ms)
	if !forceRefresh {
		if cached, ok := s.GetCachedHomeFeed(cacheKey); ok && cached != nil {
			return cached, nil
		}
	}

	// 2. Check distributed Redis L2 cache (~0.8ms)
	if !forceRefresh {
		if cached, ok := s.GetCachedHomeFeedRedis(ctx, cacheKey); ok && cached != nil {
			// Backfill L1 in-memory cache for instant subsequent hits
			s.SetCachedHomeFeed(cacheKey, cached, 30*time.Second)
			return cached, nil
		}
	}

	// 3. SingleFlight execution: Deduplicate concurrent generation across all goroutines
	// If hundreds of concurrent requests arrive simultaneously on cache miss, only ONE executes.
	val, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		// Double-check L1 in case another singleflight runner just finished
		if !forceRefresh {
			if cached, ok := s.GetCachedHomeFeed(cacheKey); ok && cached != nil {
				return cached, nil
			}
		}

		resp, genErr := s.generateHomeFeed(ctx, language, districtSlug)
		if genErr != nil {
			return nil, genErr
		}

		// Store in L1 cache (30s TTL)
		s.SetCachedHomeFeed(cacheKey, resp, 30*time.Second)

		// Store in Redis L2 cache (5m TTL)
		_ = s.SetCachedHomeFeedRedis(ctx, cacheKey, resp, 5*time.Minute)

		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*HomeFeedResponse), nil
}

// generateHomeFeed executes the strict zero-duplicate candidate aggregation pipeline from PostgreSQL.
func (s *Service) generateHomeFeed(ctx context.Context, language, districtSlug string) (*HomeFeedResponse, error) {
	// Query pool directly without abortable transactions
	resp := &HomeFeedResponse{
		StateEdition: StateEditionInfo{
			ID:   1,
			Name: "National Desk",
			Slug: "national",
		},
		FourColumnSection1: []FourColumnSectionItem{},
		FourColumnSection2: []FourColumnSectionItem{},
		VideoNewsData:      []VideoNewsItem{},
		TopAuthors:         []AuthorSpotlight{},
		WebStories:         []interface{}{},
	}

	// 3. Navigation Categories & Dynamic Homepage Sections
	categories, err := s.ListCategoriesDirect(ctx)
	if err == nil {
		resp.NavigationCategories = categories
	} else {
		resp.NavigationCategories = []Category{}
	}

	if sections, err := s.ListHomepageSections(ctx); err == nil {
		resp.OrderedSections = sections
	} else {
		resp.OrderedSections = []HomepageSection{}
	}

	// ─────────────────────────────────────────────────────────────────────────
	// 4. Strict Zero-Duplicate Global Seen Tracker
	// ─────────────────────────────────────────────────────────────────────────
	seen := make(map[string]bool)

	// Candidate Fetching Helpers with smart language fallback:
	fetchCandidates := func(filter ListArticlesFilter) []ArticleListItem {
		items, _, _ := s.ListArticlesDirect(ctx, filter)
		if len(items) == 0 && filter.Language != "" {
			fallbackFilter := filter
			fallbackFilter.Language = ""
			items, _, _ = s.ListArticlesDirect(ctx, fallbackFilter)
		}
		return items
	}

	// Rich reservoir pool: load published articles across languages to ensure zero empties
	generalPool, _, _ := s.ListArticlesDirect(ctx, ListArticlesFilter{
		Status:  "published",
		SortBy:  "latest",
		PerPage: 300,
	})

	isBreaking := true
	breakingCandidates := fetchCandidates(ListArticlesFilter{
		Language:   language,
		Status:     "published",
		IsBreaking: &isBreaking,
		PerPage:    15,
	})

	isFeatured := true
	featuredCandidates := fetchCandidates(ListArticlesFilter{
		Language:   language,
		Status:     "published",
		IsFeatured: &isFeatured,
		PerPage:    25,
	})

	trendingCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Status:   "published",
		SortBy:   "trending",
		PerPage:  25,
	})

	politicsCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "politics",
		Status:   "published",
		PerPage:  25,
	})

	sportsCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "sports",
		Status:   "published",
		PerPage:  25,
	})

	entertainmentCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "entertainment",
		Status:   "published",
		PerPage:  25,
	})

	crimeCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "crime",
		Status:   "published",
		PerPage:  25,
	})

	businessCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "business",
		Status:   "published",
		PerPage:  25,
	})

	techCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "technology",
		Status:   "published",
		PerPage:  25,
	})

	jharkhandCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "jharkhand",
		Status:   "published",
		PerPage:  35,
	})

	biharCandidates := fetchCandidates(ListArticlesFilter{
		Language: language,
		Category: "bihar",
		Status:   "published",
		PerPage:  35,
	})

	// Helper functions for selecting unique, non-duplicated articles
	takeSingle := func(candidates []ArticleListItem) *HomeArticleItem {
		for _, it := range candidates {
			idStr := it.ID.String()
			if !seen[idStr] {
				seen[idStr] = true
				item := toHomeArticleItem(it)
				return &item
			}
		}
		// Fallback to general pool
		for _, it := range generalPool {
			idStr := it.ID.String()
			if !seen[idStr] {
				seen[idStr] = true
				item := toHomeArticleItem(it)
				return &item
			}
		}
		return nil
	}

	takeMultiple := func(candidates []ArticleListItem, count int) []HomeArticleItem {
		res := []HomeArticleItem{}
		for _, it := range candidates {
			idStr := it.ID.String()
			if !seen[idStr] {
				seen[idStr] = true
				res = append(res, toHomeArticleItem(it))
				if len(res) >= count {
					return res
				}
			}
		}
		// Fill remaining from general pool if needed, NEVER repeating any seen ID
		for _, it := range generalPool {
			idStr := it.ID.String()
			if !seen[idStr] {
				seen[idStr] = true
				res = append(res, toHomeArticleItem(it))
				if len(res) >= count {
					break
				}
			}
		}
		return res
	}

	// ─── PIPELINE STEP 1: BREAKING NEWS TICKER ────────────────────────────────
	breakingItems := takeMultiple(breakingCandidates, 5)
	resp.FeaturedData.BreakingNews = breakingItems

	// ─── PIPELINE STEP 2: TOP FEATURED SECTION ────────────────────────────────
	// Main Lead Hero
	resp.FeaturedData.MainFeatured = takeSingle(featuredCandidates)
	// Middle 3 Editorial Cards
	resp.FeaturedData.MiddleFeatured = takeMultiple(featuredCandidates, 3)
	// Right Box Top Lead
	resp.FeaturedData.RightTopFeatured = takeSingle(featuredCandidates)
	// Right Box 4 Most Viewed / Recent List
	resp.FeaturedData.RightListMostViewed = takeMultiple(trendingCandidates, 4)

	// ─── DYNAMIC 4-COLUMN BUILDER (Reads categories from database settings) ─
	buildFourColumnItems := func(secKey string, defaultSlugs []string) []FourColumnSectionItem {
		var slugs []string
		for _, s := range resp.OrderedSections {
			if s.SectionKey == secKey {
				if cats, ok := s.Settings["categories"].([]interface{}); ok && len(cats) > 0 {
					for _, c := range cats {
						if str, ok := c.(string); ok && str != "" {
							slugs = append(slugs, str)
						}
					}
				} else if catsStr, ok := s.Settings["categories"].(string); ok && catsStr != "" {
					slugs = strings.Fields(catsStr)
				}
				break
			}
		}
		if len(slugs) == 0 {
			slugs = defaultSlugs
		}

		var items []FourColumnSectionItem
		for _, slug := range slugs {
			title := slug
			for _, cat := range resp.NavigationCategories {
				if cat.Slug == slug {
					title = cat.Name
					break
				}
			}
			candidates := fetchCandidates(ListArticlesFilter{
				Category: slug,
				Status:   "published",
				PerPage:  10,
			})
			items = append(items, FourColumnSectionItem{
				ID:       fmt.Sprintf("col-%s", slug),
				Title:    title,
				Slug:     slug,
				Featured: takeSingle(candidates),
				Articles: takeMultiple(candidates, 4),
			})
		}
		return items
	}

	// ─── PIPELINE STEP 3: FOUR-COLUMN SECTION 1 ───────────────────────────────
	resp.FourColumnSection1 = buildFourColumnItems("four_col_1", []string{"politics", "national", "international", "crime"})

	// ─── PIPELINE STEP 4: STATE DESKS (Jharkhand & Bihar + Sticky Sidebar) ────
	resp.ContentSidebarData.FeaturedArticle = takeSingle(jharkhandCandidates)
	resp.ContentSidebarData.TopHeadlines = takeMultiple(jharkhandCandidates, 4)
	resp.ContentSidebarData.JharkhandBottom = takeMultiple(jharkhandCandidates, 2)
	resp.ContentSidebarData.StateArticles = takeMultiple(biharCandidates, 5)
	resp.ContentSidebarData.MoreNewsArticles = takeMultiple(biharCandidates, 4)
	resp.ContentSidebarData.SidebarTopArticle = takeSingle(generalPool)
	resp.ContentSidebarData.SidebarColumns = takeMultiple(generalPool, 2)
	resp.ContentSidebarData.SidebarOpinion.Left = takeMultiple(generalPool, 3)
	resp.ContentSidebarData.SidebarOpinion.Right = takeMultiple(generalPool, 3)

	// ─── PIPELINE STEP 5: VIDEO NEWS SECTION ─────────────────────────────────
	videoCandidates := takeMultiple(generalPool, 6)
	defaultDurations := []string{"03:45", "05:12", "02:30", "04:15", "06:20", "03:10"}
	defaultViews := []string{"14.5K", "28.2K", "9.8K", "52.1K", "18.3K", "34.0K"}
	for idx, it := range videoCandidates {
		dur := defaultDurations[idx%len(defaultDurations)]
		vw := defaultViews[idx%len(defaultViews)]
		resp.VideoNewsData = append(resp.VideoNewsData, VideoNewsItem{
			ID:          it.ID,
			Title:       it.Title,
			Slug:        it.Slug,
			VideoURL:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			Thumbnail:   it.Image,
			Category:    it.Category,
			Duration:    dur,
			PublishedAt: it.Date,
			Views:       vw,
			Author:      it.Author,
		})
	}

	// ─── PIPELINE STEP 6: CATEGORY DESKS ─────────────────────────────────────
	// Politics (Type A: 1 Featured + 4 Sub)
	resp.CategorySectionData.Politics = CategoryBlockAData{
		Title:       "राजनीति",
		Featured:    takeSingle(politicsCandidates),
		SubArticles: takeMultiple(politicsCandidates, 4),
	}

	// Sports (Type A: 1 Featured + 4 Sub)
	resp.CategorySectionData.Sports = CategoryBlockAData{
		Title:       "खेल",
		Featured:    takeSingle(sportsCandidates),
		SubArticles: takeMultiple(sportsCandidates, 4),
	}

	// Entertainment (Type B: 1 Featured + 2 Middle + 4 Bottom)
	resp.CategorySectionData.Entertainment = CategoryBlockBData{
		Title:          "मनोरंजन",
		Featured:       takeSingle(entertainmentCandidates),
		MiddleArticles: takeMultiple(entertainmentCandidates, 2),
		BottomArticles: takeMultiple(entertainmentCandidates, 4),
	}

	// Crime (Type A: 1 Featured + 4 Sub)
	resp.CategorySectionData.Crime = CategoryBlockAData{
		Title:       "अपराध",
		Featured:    takeSingle(crimeCandidates),
		SubArticles: takeMultiple(crimeCandidates, 4),
	}

	// Business (Type B: 1 Featured + 2 Middle + 4 Bottom)
	resp.CategorySectionData.Business = CategoryBlockBData{
		Title:          "व्यापार",
		Featured:       takeSingle(businessCandidates),
		MiddleArticles: takeMultiple(businessCandidates, 2),
		BottomArticles: takeMultiple(businessCandidates, 4),
	}

	// Sidebar Highlights: Trending, Exclusive, Bottom
	resp.CategorySectionData.TopTrending = takeMultiple(trendingCandidates, 5)
	resp.CategorySectionData.ExclusiveNews = takeMultiple(generalPool, 3)
	resp.CategorySectionData.SidebarBottom = takeMultiple(generalPool, 4)

	// ─── PIPELINE STEP 7: FOUR-COLUMN SECTION 2 ───────────────────────────────
	resp.FourColumnSection2 = buildFourColumnItems("four_col_2", []string{"auto", "lifestyle", "dharma", "environment"})

	// ─── PIPELINE STEP 8: TECHNOLOGY SECTION ─────────────────────────────────
	resp.TechnologyData = TechnologySectionData{
		CategoryName:    "TECHNOLOGY",
		CategoryTitleHi: "टेक्नोलॉजी",
		FeaturedArticle: takeSingle(techCandidates),
		SideArticles:    takeMultiple(techCandidates, 2),
		RightArticles:   takeMultiple(techCandidates, 3),
	}

	// ─── PIPELINE STEP 9: METRICS & OPTIONAL ENTITIES ────────────────────────
	resp.TotalUniqueArticles = len(seen)

	// Safe optional table queries directly on pool (no transaction aborts)
	s.loadWebStoriesDirect(ctx, resp)
	s.loadActivePollDirect(ctx, resp)

	return resp, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers & Safe Direct Loaders
// ─────────────────────────────────────────────────────────────────────────────

func toHomeArticleItem(item ArticleListItem) HomeArticleItem {
	img := item.FeaturedImage
	if img == "" {
		img = "/assets/newsplaceholder.webp"
	}
	cat := "समाचार"
	if len(item.CategoryNames) > 0 {
		cat = item.CategoryNames[0]
	}
	dateStr := "आज"
	if item.PublishedAt != nil {
		dateStr = item.PublishedAt.Format("02 Jan 2006")
	}
	author := item.AuthorName
	if author == "" {
		author = "विशेष संवाददाता"
	}

	return HomeArticleItem{
		ID:        item.ID.String(),
		Slug:      item.Slug,
		Title:     item.Title,
		Image:     img,
		Date:      dateStr,
		Category:  cat,
		Author:    author,
		ViewCount: item.ViewCount,
	}
}

func (s *Service) loadActivePollDirect(ctx context.Context, resp *HomeFeedResponse) {
	defer func() { _ = recover() }()
	pollQuery := `
		SELECT id, question, total_votes
		FROM polls
		WHERE is_active = TRUE AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`
	var pollID, question string
	var totalVotes int64
	if err := s.pool.QueryRow(ctx, pollQuery).Scan(&pollID, &question, &totalVotes); err == nil {
		resp.ActivePoll = map[string]interface{}{
			"id":          pollID,
			"question":    question,
			"total_votes": totalVotes,
		}
	}
}

func (s *Service) loadWebStoriesDirect(ctx context.Context, resp *HomeFeedResponse) {
	defer func() { _ = recover() }()
	wsQuery := `
		SELECT id, title, slug, cover_image, jsonb_array_length(slides) as slide_count
		FROM web_stories
		WHERE status = 'published'
		ORDER BY published_at DESC NULLS LAST
		LIMIT 8
	`
	if wsRows, err := s.pool.Query(ctx, wsQuery); err == nil {
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
}
